package repository

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pure arithmetic, so every test here runs without a database.

// expectedMedian states the contract independently of the implementation: sort
// a copy and take the lower middle, which is 0 for an empty slice.
func expectedMedian(d []time.Duration) time.Duration {
	sorted := append([]time.Duration(nil), d...)
	slices.Sort(sorted)
	if len(sorted) == 0 {
		return 0
	}
	return sorted[(len(sorted)-1)/2]
}

func TestMedianDuration(t *testing.T) {
	tests := []struct {
		name string
		in   []time.Duration
		want time.Duration
	}{
		{
			name: "empty has no median",
			in:   nil,
			want: 0,
		},
		{
			name: "single element is its own median",
			in:   []time.Duration{7 * time.Second},
			want: 7 * time.Second,
		},
		{
			name: "odd length takes the middle",
			in:   []time.Duration{5, 1, 3, 9, 7},
			want: 5,
		},
		{
			name: "even length takes the lower middle, never the average",
			in:   []time.Duration{1, 2, 3, 4},
			want: 2,
		},
		{
			name: "even length of two takes the smaller",
			in:   []time.Duration{30, 10},
			want: 10,
		},
		{
			name: "already sorted",
			in:   ascendingDurations(101),
			want: time.Duration(50),
		},
		{
			name: "reverse sorted gives the same answer as sorted",
			in:   reverseDurations(101),
			want: time.Duration(50),
		},
		{
			// 501 occurrences of 1ms and 500 of 2ms: the lower middle is still a
			// real gap, so an even-ish split cannot round up into the other value.
			name: "repeated two-value pattern",
			in:   repeatPatternDurations([]time.Duration{time.Millisecond, 2 * time.Millisecond}, 1001),
			want: time.Millisecond,
		},
		{
			name: "one huge outlier must not drag the median",
			in:   []time.Duration{time.Hour, time.Millisecond, time.Millisecond, 24 * time.Hour},
			want: time.Millisecond,
		},
		{
			name: "all negative gaps still pick the lower middle",
			in:   []time.Duration{-4, -1, -2, -3},
			want: -3,
		},
		{
			// Six 2ms against five 1ms over an odd length: the middle lands in
			// the 2ms run.
			name: "two distinct values repeated",
			in:   repeatPatternDurations([]time.Duration{2 * time.Millisecond, time.Millisecond}, 11),
			want: 2 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := medianDuration(tt.in)
			assert.Equal(t, tt.want, got)
			// The answer is an element of the input, never an average of two
			// neighbours, so re-running on a sorted copy is stable.
			assert.Equal(t, tt.want, medianDuration(append([]time.Duration(nil), tt.in...)))
		})
	}
}

// A gallery burst of 15 000 identical gaps is the realistic shape of this input:
// the camera held still between two poses. A 2-way partition peels one element
// off per pass on such an input, so this case pins the duplicate-heavy path that
// the 3-way partition exists for.
func TestMedianDurationAllEqualBulk(t *testing.T) {
	const n = 15000
	gaps := make([]time.Duration, n)
	for i := range gaps {
		gaps[i] = time.Millisecond
	}
	assert.Equal(t, time.Millisecond, medianDuration(gaps))

	// One value differs, in every position the sort could put it: the median
	// must be the repeated one.
	for _, pos := range []int{0, 1, n / 2, n - 2, n - 1} {
		gaps := make([]time.Duration, n)
		for i := range gaps {
			gaps[i] = time.Millisecond
		}
		gaps[pos] = time.Hour
		assert.Equal(t, time.Millisecond, medianDuration(gaps), "outlier at index %d", pos)
	}
}

// The order statistics of the generated shapes are not derived by hand: the
// expected value is the lower middle of a sorted copy, which is the definition
// the helper is supposed to implement. That keeps these cases honest even
// though the shapes are all about defeating a bad pivot choice.
func TestMedianDurationMatchesSortedCopy(t *testing.T) {
	tests := []struct {
		name string
		in   []time.Duration
	}{
		{name: "ascending", in: ascendingDurations(101)},
		{name: "reverse sorted", in: reverseDurations(101)},
		{name: "even length ascending", in: ascendingDurations(100)},
		{name: "single element", in: ascendingDurations(1)},
		// Every interior element is larger than both ends, so a pivot read from
		// the edges is the minimum of the window at every step.
		{name: "valley defeats an edge pivot", in: valleyDurations(2001)},
		// The mirror image: every interior element is smaller than both ends, so
		// an edge pivot is the maximum of the window at every step.
		{name: "peak defeats an edge pivot", in: peakDurations(2001)},
		{name: "organ pipe defeats an edge pivot", in: organPipeDurations(2001)},
		// Sawtooth: an edge pivot is alternately the largest and the smallest
		// remaining value, so neither side ever makes progress.
		{name: "sawtooth defeats an edge pivot", in: sawtoothDurations(2003)},
		{name: "two distinct values", in: repeatPatternDurations([]time.Duration{1, 2}, 9999)},
		{name: "three distinct values", in: repeatPatternDurations([]time.Duration{1, 2, 3}, 9999)},
		{name: "duplicate-heavy burst", in: burstDurations(15000, time.Millisecond)},
		{name: "burst with a few wider gaps", in: burstWithOutliersDurations(15000)},
		{name: "interleaved with duplicates", in: repeatPatternDurations([]time.Duration{5, 5, 5, 1}, 999)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := expectedMedian(tt.in)
			assert.Equal(t, want, medianDuration(tt.in))
			// Re-running on an untouched copy must give the same answer, so the
			// helper depends on no state left behind by a previous call.
			assert.Equal(t, want, medianDuration(append([]time.Duration(nil), tt.in...)))
		})
	}
}

// The ordering of the input is destroyed on purpose, but the multiset is not:
// the caller must be able to rely on nothing else changing.
func TestMedianDurationPreservesContents(t *testing.T) {
	for _, in := range [][]time.Duration{
		valleyDurations(999),
		peakDurations(999),
		sawtoothDurations(999),
		burstWithOutliersDurations(1000),
	} {
		sorted := append([]time.Duration(nil), in...)
		slices.Sort(sorted)

		got := medianDuration(in)
		slices.Sort(in) // the call permuted the slice; the values are unchanged

		assert.Equal(t, expectedMedian(sorted), got)
		assert.Equal(t, sorted, in, "the multiset must survive the in-place selection")
	}
}

func TestMedianDurationDoesNotAllocate(t *testing.T) {
	// Pre-built scratch input: quickselect must not copy the slice it works on.
	gaps := valleyDurations(4099)
	allocs := testing.AllocsPerRun(10, func() {
		medianDuration(gaps)
	})
	assert.Zero(t, allocs, "medianDuration must stay allocation-free")
}

// A quadratic implementation cannot finish these sizes inside the budget; the
// point is the runtime, not the returned value (correctness is asserted
// separately). Each shape is chosen so that a bad pivot choice makes one pass
// discard a single element.
func TestMedianDurationStaysLinearOnAdversarialInput(t *testing.T) {
	const n = 300000
	shapes := []struct {
		name string
		in   []time.Duration
	}{
		{"already sorted", ascendingDurations(n)},
		{"reverse sorted", reverseDurations(n)},
		{"valley", valleyDurations(n)},
		{"peak", peakDurations(n)},
		{"organ pipe", organPipeDurations(n)},
		{"sawtooth", sawtoothDurations(n)},
		{"two distinct values", repeatPatternDurations([]time.Duration{1, 2}, n)},
		{"three distinct values", repeatPatternDurations([]time.Duration{1, 2, 3}, n)},
		{"duplicate-heavy burst", burstDurations(n, time.Millisecond)},
		{"burst with wider gaps", burstWithOutliersDurations(n)},
	}

	for _, shape := range shapes {
		t.Run(shape.name, func(t *testing.T) {
			want := expectedMedian(shape.in)
			start := time.Now()
			got := medianDuration(shape.in)
			elapsed := time.Since(start)
			assert.Equal(t, want, got)
			assert.Less(t, elapsed, 5*time.Second, "quadratic behaviour on n=%d", n)
		})
	}
}

func TestSpreadEvenly(t *testing.T) {
	tests := []struct {
		name string
		ids  []int
		k    int
		want []int
	}{
		{name: "zero budget", ids: []int{1, 2, 3}, k: 0, want: nil},
		{name: "negative budget", ids: []int{1, 2, 3}, k: -1, want: nil},
		{name: "empty input", ids: nil, k: 4, want: nil},
		{name: "budget below one", ids: []int{5}, k: 0, want: nil},
		{name: "budget of one takes the first element", ids: []int{5, 9, 13}, k: 1, want: []int{5}},
		{name: "budget of one on empty input", ids: nil, k: 1, want: nil},
		{name: "two elements with budget two", ids: []int{4, 8}, k: 2, want: []int{4, 8}},
		{name: "budget equals length returns everything", ids: []int{1, 2, 3}, k: 3, want: []int{1, 2, 3}},
		{name: "budget above length returns everything", ids: []int{1, 2, 3}, k: 99, want: []int{1, 2, 3}},
		{name: "two elements, budget one", ids: []int{4, 8}, k: 1, want: []int{4}},
		{name: "four elements, budget two pins both ends", ids: []int{1, 2, 3, 4}, k: 2, want: []int{1, 4}},
		{name: "four elements, budget three", ids: []int{1, 2, 3, 4}, k: 3, want: []int{1, 3, 4}},
		{name: "five elements, budget three rounds to the middle", ids: []int{10, 20, 30, 40, 50}, k: 3, want: []int{10, 30, 50}},
		{
			name: "five elements, budget four interpolates both interior slots",
			ids:  []int{10, 20, 30, 40, 50},
			k:    4,
			want: []int{10, 20, 40, 50},
		},
		{
			// Indices 1.5 and 4.5 round away from zero to 2 and 5.
			name: "half steps round away from zero",
			ids:  []int{0, 10, 20, 30, 40, 50, 60},
			k:    5,
			want: []int{0, 20, 30, 50, 60},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := append([]int(nil), tt.ids...)
			got := spreadEvenly(ids, tt.k)
			assert.Equal(t, tt.want, got)
			// The caller's slice is never reordered or resized.
			assert.Equal(t, tt.ids, ids)
		})
	}
}

// The invariants the SQL depends on: exactly k offsets, strictly increasing, and
// both ends of the domain present so the strip cannot silently drop the newest
// photos.
func TestSpreadEvenlySpansTheDomain(t *testing.T) {
	for n := 2; n <= 64; n++ {
		ids := make([]int, n)
		for i := range ids {
			ids[i] = i * 3
		}
		for k := 2; k <= n; k++ {
			got := spreadEvenly(append([]int(nil), ids...), k)
			require.Len(t, got, k, "n=%d k=%d", n, k)
			assert.Equal(t, 0, got[0], "n=%d k=%d: first element must be pinned", n, k)
			assert.Equal(t, (n-1)*3, got[k-1], "n=%d k=%d: last element must be pinned", n, k)
			for i := 1; i < k; i++ {
				assert.Less(t, got[i-1], got[i], "n=%d k=%d: offsets must increase at %d", n, k, i)
				assert.Contains(t, ids, got[i], "n=%d k=%d: offset must come from the input", n, k)
			}
		}
	}
}

// The result of an over-large budget is a copy: callers may append to it.
func TestSpreadEvenlyOverBudgetReturnsCopy(t *testing.T) {
	ids := []int{1, 2, 3}
	got := spreadEvenly(ids, 10)
	require.Equal(t, ids, got)
	got[0] = 99
	assert.Equal(t, []int{1, 2, 3}, ids)
}

func TestSaturatingMul(t *testing.T) {
	const maxDur = time.Duration(math.MaxInt64)

	tests := []struct {
		name string
		a    time.Duration
		b    int64
		want time.Duration
	}{
		{name: "ordinary product", a: 3 * time.Second, b: 10, want: 30 * time.Second},
		{name: "product of one", a: 42 * time.Hour, b: 1, want: 42 * time.Hour},
		{name: "zero multiplier", a: 3 * time.Second, b: 0, want: 0},
		{name: "zero operand", a: 0, b: 10, want: 0},
		{name: "negative operand", a: -3 * time.Second, b: 10, want: 0},
		{name: "negative multiplier", a: 3 * time.Second, b: -10, want: 0},
		{
			name: "largest value scaled by one",
			a:    maxDur,
			b:    1,
			want: maxDur,
		},
		{
			// Integer division of the boundary, so the largest operand whose
			// product still fits leaves 7ns of headroom.
			name: "exactly at the boundary",
			a:    maxDur / 10,
			b:    10,
			want: maxDur - 7,
		},
		{
			name: "one past the boundary clamps",
			a:    maxDur/10 + 1,
			b:    10,
			want: maxDur,
		},
		{
			name: "one times the largest multiplier is exact",
			a:    1,
			b:    math.MaxInt64,
			want: maxDur,
		},
		{
			name: "two times the largest multiplier clamps",
			a:    2,
			b:    math.MaxInt64,
			want: maxDur,
		},
		{
			name: "largest value times two clamps",
			a:    maxDur,
			b:    2,
			want: maxDur,
		},
		{
			name: "overflowing product of three clamps",
			a:    time.Duration(math.MaxInt64/3) + 1,
			b:    3,
			want: maxDur,
		},
		{
			// 250 years is inside the range of time.Duration but not inside
			// what that many decades of seconds times ten can hold.
			name: "a two-hundred-fifty-year gap scaled by ten clamps",
			a:    250 * 365 * 24 * time.Hour,
			b:    10,
			want: maxDur,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, saturatingMul(tt.a, tt.b))
		})
	}
}

// The bug this helper exists for: a naive `median * 10` on a large enough gap
// wraps to a negative duration, and a negative threshold is beaten by every
// gap, which silently switches the isolation pass off.
func TestSaturatingMulNeverReturnsNegative(t *testing.T) {
	a := time.Duration(math.MaxInt64/10) + 1
	require.Negative(t, a*10, "precondition: the naive product wraps negative")

	got := saturatingMul(a, 10)
	assert.Positive(t, got, "a wrapped threshold would make every gap look isolated")
	assert.Equal(t, time.Duration(math.MaxInt64), got)
	assert.Greater(t, got, a, "a saturated threshold must exceed any real gap")
}

func ascendingDurations(n int) []time.Duration {
	d := make([]time.Duration, n)
	for i := range d {
		d[i] = time.Duration(i)
	}
	return d
}

func reverseDurations(n int) []time.Duration {
	d := make([]time.Duration, n)
	for i := range d {
		d[i] = time.Duration(n - 1 - i)
	}
	return d
}

func valleyDurations(n int) []time.Duration {
	d := make([]time.Duration, n)
	mid := n / 2
	for i := range d {
		if i <= mid {
			d[i] = time.Duration(mid - i)
		} else {
			d[i] = time.Duration(i - mid)
		}
	}
	return d
}

func peakDurations(n int) []time.Duration {
	d := make([]time.Duration, n)
	mid := n / 2
	for i := range d {
		if i <= mid {
			d[i] = time.Duration(i)
		} else {
			d[i] = time.Duration(n - 1 - i)
		}
	}
	return d
}

func sawtoothDurations(n int) []time.Duration {
	d := make([]time.Duration, n)
	for i := range d {
		if i%2 == 0 {
			d[i] = time.Duration(i)
		} else {
			d[i] = time.Duration(n - i)
		}
	}
	return d
}

func organPipeDurations(n int) []time.Duration {
	d := make([]time.Duration, n)
	mid := n / 2
	for i := range d {
		dist := i
		if i > mid {
			dist = n - 1 - i
		}
		d[i] = time.Duration(dist)
	}
	return d
}

func repeatPatternDurations(pattern []time.Duration, n int) []time.Duration {
	d := make([]time.Duration, n)
	for i := range d {
		d[i] = pattern[i%len(pattern)]
	}
	return d
}

// burstDurations is the shape the caller really passes: a long run of one
// repeated gap, which is what a camera holding still between two poses records.
func burstDurations(n int, gap time.Duration) []time.Duration {
	d := make([]time.Duration, n)
	for i := range d {
		d[i] = gap
	}
	return d
}

// burstWithOutliersDurations keeps the duplicate-heavy majority but scatters
// wider gaps through it, so the 3-way partition is exercised on an input that
// has both a huge equal run and real ordering work left to do.
func burstWithOutliersDurations(n int) []time.Duration {
	d := burstDurations(n, time.Millisecond)
	for i := 0; i < n; i += 977 {
		d[i] = time.Duration(i%7+1) * time.Hour
	}
	return d
}
