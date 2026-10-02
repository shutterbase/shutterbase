package repository

import (
	"math"
	"math/bits"
	"slices"
	"time"
)

// Pure numeric helpers behind the time-tick strip: a representative gap, a set
// of evenly spaced row offsets to probe with, and a saturation-safe scale of the
// gap. They live here, next to the SQL that consumes their results, because
// every number they produce is a threshold rather than a measurement — a
// threshold that overflows or that averages two samples which never occurred
// silently changes which photos are treated as isolated.

// medianSortCutoff is the sub-range size below which partitioning stops paying
// for itself and a direct sort wins. Its only job is to keep the constant factor
// low for the short ranges quickselect keeps descending into; correctness does
// not depend on the value.
const medianSortCutoff = 16

// medianDuration returns a representative gap from d, choosing the lower of the
// two middle values when the length is even so that the result is always an
// element of d and never an invented average. That matters because the caller
// multiplies the result by a factor and compares it against real gaps: a median
// of 3s and 4s would be 3.5s, which matches no observed spacing at all, and the
// "isolated photo" decision made from it would rest on a value that does not
// exist in the data.
//
// Quickselect, not sort.Slice: a gallery burst is around 15 000 rows, and only
// the middle one is ever read, so paying the full O(n log n) ordering to discard
// it is wasted work on a request path that already scanned the same rows. The
// partition is 3-way (Dutch national flag) because the realistic input for this
// call is extremely duplicate-heavy — 15 000 identical gaps when the camera held
// still between two poses — and a 2-way split makes every partition of such an
// input peel off a single element, which is exactly the O(n^2) / stack-exhaustion
// case the burst data produces.
//
// Allocation: none. The input slice is PERMUTED IN PLACE as a side effect — the
// contents are preserved but the order is destroyed, which quickselect cannot
// avoid without copying. Callers must therefore pass a scratch slice they built
// for this purpose (image.go builds a throwaway gaps slice) and must not hand
// over a slice whose ordering they still depend on. Sorting in place is what
// keeps the promise of zero allocations.
//
// An empty slice has no median and yields 0.
func medianDuration(d []time.Duration) time.Duration {
	if len(d) == 0 {
		return 0
	}
	// Index rather than rank: for an even length this lands on the lower middle.
	k := (len(d) - 1) / 2
	// Each successful partition cuts the range by a constant factor, so ~log2(n)
	// of them suffice. Two bits per level leaves room for the unlucky skew that
	// a median-of-three pivot can still produce before the budget is spent.
	return selectDuration(d, 0, len(d)-1, k, 2*bits.Len(uint(len(d))))
}

// selectDuration places the k-th smallest element of d at index k, where k is
// always inside [lo,hi] and every element outside that window is already known
// to be on the correct side of it. That invariant is what makes the direct
// d[k] returns correct after the range has been reordered, and it is why depth
// budget rather than a stack-depth check is what bounds the recursion.
func selectDuration(d []time.Duration, lo, hi, k, depth int) time.Duration {
	for hi-lo+1 > medianSortCutoff {
		if depth <= 0 {
			// The pivot chain is degenerating. Finishing with an in-place sort
			// caps the work at O(m log m) for the remaining m instead of letting
			// a bad input run quadratic. The window invariant still holds after
			// the sort, so d[k] is the global order statistic.
			slices.Sort(d[lo : hi+1])
			return d[k]
		}
		depth--

		lt, gt := partitionDuration(d, lo, hi, pivotDuration(d, lo, hi))
		switch {
		case k < lt:
			hi = lt - 1
		case k <= gt:
			// k landed in the run of elements equal to the pivot, which is where
			// duplicate-heavy input spends most of its time: no further recursion.
			return d[k]
		default:
			lo = gt + 1
		}
	}
	slices.Sort(d[lo : hi+1])
	return d[k]
}

// pivotDuration picks the median of the first, middle and last element of the
// window as the split value. Median-of-three costs a constant amount and stops
// the two cheap adversaries — already-sorted and reverse-sorted input, which is
// what a gallery's gap vector usually is — from degenerating immediately.
func pivotDuration(d []time.Duration, lo, hi int) time.Duration {
	mid := lo + (hi-lo)/2
	a, b, c := d[lo], d[mid], d[hi]
	if a > b {
		a, b = b, a
	}
	if b > c {
		b = c
	}
	if a > b {
		b = a
	}
	return b
}

// partitionDuration orders d[lo:hi+1] around pivot and returns the bounds of
// the run equal to it: everything below lt is smaller than pivot, everything in
// [lt,gt] is equal, everything above gt is larger. Equal values are neither
// moved nor re-examined, which is what keeps the all-equal burst case linear
// rather than merely correct.
func partitionDuration(d []time.Duration, lo, hi int, pivot time.Duration) (int, int) {
	lt, i, gt := lo, lo, hi
	for i <= gt {
		switch {
		case d[i] < pivot:
			d[lt], d[i] = d[i], d[lt]
			lt++
			i++
		case d[i] > pivot:
			d[i], d[gt] = d[gt], d[i]
			gt--
		default:
			i++
		}
	}
	return lt, gt
}

// spreadEvenly returns at most k of ids chosen by even spacing over the SLICE
// INDEX, not over the timestamps. The ids come back already ordered by capture
// time, so index spacing is also time spacing and the sample covers the whole
// domain evenly without the caller having to know how the timeline is shaped.
//
// The first and last entries are pinned whenever k >= 2, so the strip always
// reaches both ends of the range. Losing the right edge is a bug that has
// already shipped once in this code path: the newest photos were dropped
// because the sample stopped short of them.
//
// k <= 0 and an empty input both yield nil; asking for at least as many entries
// as exist returns a copy of all of them, so the caller can hand the result
// straight to the statement without aliasing its own slice; k == 1 returns only
// ids[0], because a single sample has no interior to distribute. The returned
// entries are strictly increasing whenever len(ids) > k >= 2, which is what
// makes an ordered result over a tie-heavy timeline unambiguous.
func spreadEvenly(ids []int, k int) []int {
	if k <= 0 || len(ids) == 0 {
		return nil
	}
	if k >= len(ids) {
		return slices.Clone(ids)
	}
	if k == 1 {
		return []int{ids[0]}
	}

	out := make([]int, k)
	out[0] = ids[0]
	out[k-1] = ids[len(ids)-1]

	// Interior positions are interpolated between the two pinned ends. The
	// divisor is k-1, not k: the endpoints are already spent, so the interior
	// slots k-2 divide the gap between them into k-2 equal steps. Rounding
	// half away from zero keeps the first interior entry clear of ids[0] and the
	// last clear of ids[len-1].
	span := len(ids) - 1
	steps := k - 1
	for i := 1; i < k-1; i++ {
		idx := int(math.Round(float64(i) * float64(span) / float64(steps)))
		out[i] = ids[idx]
	}
	return out
}

// saturatingMul returns a*b, or math.MaxInt64 when that product does not fit.
// It exists for `median * 10`, the isolation threshold in the time-tick strip:
// a median gap large enough to overflow that multiplication wraps to a NEGATIVE
// duration, and every gap then compares as larger than the threshold, so the
// strip marks photos isolated when nothing is.
//
// Saturating instead of wrapping keeps the threshold on the safe side of the
// comparison: clamped to MaxInt64 it exceeds every real gap, which costs at most
// the redundant isolation pass, whereas wrapping produces a threshold no input
// can exceed and silently disables the pass entirely.
//
// b must be >= 0; a non-positive multiplier or operand yields 0.
func saturatingMul(a time.Duration, b int64) time.Duration {
	if a <= 0 || b <= 0 {
		return 0
	}
	// Integer division makes this an exact overflow test: a*b fits exactly when
	// a is no greater than MaxInt64/b, and doing it this way avoids the
	// multiply-then-check that wraps before it can be noticed.
	if a > time.Duration(math.MaxInt64/b) {
		return time.Duration(math.MaxInt64)
	}
	return a * time.Duration(b)
}
