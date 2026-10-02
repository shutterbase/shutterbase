package seed

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// drawExtraTags resamples until it has n DISTINCT pool entries. That loop has no
// exit when n > len(pool) — it spins forever — and its rng.Intn(len(pool)) panics
// outright on an empty pool. Both are reachable from the loaders: the pool is
// whatever the manifest/EnsureTagSet produced, and n comes from --tag-count, so
// `--tag-count 5` against a three-tag file never returns and an emptied pool
// panics mid-chunk.
//
// Every case therefore runs behind an explicit timeout: the alternative is a
// `go test` that hangs until the package timeout and reports nothing about which
// input spun.

// drawTimeout is generous (a correct draw is microseconds) but finite.
const drawTimeout = 10 * time.Second

// drawOutcome carries either the drawn tags or the panic value, so a panicking
// input is an assertion failure rather than a crashed test binary.
type drawOutcome struct {
	tags  []string
	panic any
}

// within runs draw on its own goroutine and returns what it produced. A panic is
// recovered there so it is reported per case; a hang is fatal to the case.
func within(t *testing.T, draw func() []string) drawOutcome {
	t.Helper()
	done := make(chan drawOutcome, 1)
	go func() {
		out := drawOutcome{}
		defer func() {
			if r := recover(); r != nil {
				out = drawOutcome{panic: r}
			}
			done <- out
		}()
		out.tags = draw()
	}()
	select {
	case got := <-done:
		return got
	case <-time.After(drawTimeout):
		t.Fatalf("draw did not return within %s: the resample loop spins forever when n > len(pool)", drawTimeout)
		return drawOutcome{}
	}
}

func pool10() []string {
	pool := make([]string, 10)
	for i := range pool {
		pool[i] = fmt.Sprintf("Tag%02d", i)
	}
	return pool
}

// The bounds, not the distribution. A draw must return exactly min(n, len(pool))
// DISTINCT entries from the pool, and must return for every input — including the
// two that used to hang or panic.
func TestDrawExtraTagsBounds(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pool    []string
		n       int
		wantLen int
	}{
		{"n below the pool size", pool10(), 4, 4},
		{"n equal to the pool size", pool10(), 10, 10},
		{"n above the pool size", pool10(), 13, 10},
		{"zero draw", pool10(), 0, 0},
		{"empty pool", nil, 3, 0},
		{"empty pool zero draw", nil, 0, 0},
		{"empty pool one tag requested", []string{}, 1, 0},
		{"one tag pool drawn twice", []string{"Tag00"}, 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool, n := tc.pool, tc.n
			got := within(t, func() []string { return drawExtraTags(rand.New(rand.NewSource(1)), pool, n) })

			require.Nil(t, got.panic,
				"drawExtraTags panicked (%v) on a pool of %d entries asking for %d", got.panic, len(pool), n)
			require.Len(t, got.tags, tc.wantLen,
				"n=%d over a pool of %d must yield min(n, len(pool)) = %d distinct tags, got %v",
				n, len(pool), tc.wantLen, got.tags)

			seen := map[string]bool{}
			for _, tag := range got.tags {
				assert.False(t, seen[tag], "tag %q drawn twice", tag)
				seen[tag] = true
				assert.Contains(t, pool, tag, "tag %q is not in the pool", tag)
			}
		})
	}
}

// A full pool asked for one more than it holds must return the whole pool, not
// spin: that is the `--tag-count` above the tag-file length case.
func TestDrawExtraTagsReturnsTheWholePoolWhenAskedForMore(t *testing.T) {
	pool := []string{"Car", "Track", "Podium"}
	got := within(t, func() []string { return drawExtraTags(rand.New(rand.NewSource(1)), pool, 5) })
	require.Nil(t, got.panic, "drawExtraTags panicked (%v)", got.panic)
	assert.ElementsMatch(t, pool, got.tags, "n above the pool size must yield the whole pool")
}

// The pinned-count entry point is the realistic way to reach the over-large n:
// --tag-count N goes straight into drawExtraTags with no cap.
func TestPhotoExtrasPinnedCountAboveThePoolDoesNotHang(t *testing.T) {
	pool := []string{"Car", "Track", "Podium"}
	got := within(t, func() []string { return photoExtrasFixed(rand.New(rand.NewSource(1)), pool, 5) })
	require.Nil(t, got.panic, "photoExtrasFixed panicked (%v)", got.panic)
	assert.ElementsMatch(t, pool, got.tags, "a pinned count above the pool size must yield the whole pool")

	// The documented draw (count 0) over a pool smaller than the largest count in
	// the 30/50/20 split has the same problem, so it is pinned too.
	drawn := within(t, func() []string { return photoExtras(rand.New(rand.NewSource(1)), pool) })
	require.Nil(t, drawn.panic, "photoExtras panicked (%v)", drawn.panic)
	assert.Subset(t, pool, drawn.tags)
	assert.LessOrEqual(t, len(drawn.tags), len(pool), "a photo can never carry more tags than the pool holds")
}

// A fixed source must still give a fixed draw — the loader's idempotency rests on
// the per-index seed reproducing the same tag set on every re-run.
func TestDrawExtraTagsIsDeterministicForAFixedSeed(t *testing.T) {
	pool := pool10()
	first := within(t, func() []string { return drawExtraTags(rand.New(rand.NewSource(1)), pool, 3) })
	second := within(t, func() []string { return drawExtraTags(rand.New(rand.NewSource(1)), pool, 3) })
	require.Nil(t, first.panic)
	require.Nil(t, second.panic)
	require.Len(t, first.tags, 3)
	assert.Equal(t, first.tags, second.tags, "the same seed must reproduce the same draw")

	// And the whole pool stays reachable across streams, or a fixed seed would be
	// pinning a subset of the vocabulary to every photo.
	used := map[string]bool{}
	for i := range 500 {
		got := drawExtraTags(rand.New(rand.NewSource(int64(i))), pool, 3)
		require.Len(t, got, 3)
		for _, tag := range got {
			used[tag] = true
		}
	}
	assert.Len(t, used, len(pool), "only %d of %d pool tags are reachable: %v", len(used), len(pool), used)
}
