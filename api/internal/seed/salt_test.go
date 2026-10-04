package seed

import (
	"testing"
)

// --seed folds into every draw the loaders make. This matters because the draw is
// ALREADY reproducible without it: the per-photo seeds come from the photo's
// index and its image id, not from a wall clock, which is what makes a re-run
// idempotent. The salt exists for the other case — two DIFFERENT fixture sets
// from the same command, so a reviewer can compare them side by side.
//
// These are the unsalted forms the loader tests pin, and the salted ones must
// still be a pure function of their inputs: no global state, no package-level rng.

func TestSaltedSeedsArePureFunctions(t *testing.T) {
	// Repeated calls agree — a draw that advanced a shared counter would make a
	// re-run produce different photos, which is the defect the per-index seeding
	// work removed.
	for i := range 200 {
		if got, want := saltedIndexSeed("W", i, 42), saltedIndexSeed("W", i, 42); got != want {
			t.Fatalf("saltedIndexSeed(\"W\", %d, 42) not stable: %d then %d", i, want, got)
		}
		if got, want := saltedBurstSeed(i, i%7, 42), saltedBurstSeed(i, i%7, 42); got != want {
			t.Fatalf("saltedBurstSeed(%d, …, 42) not stable: %d then %d", i, want, got)
		}
		if got, want := saltedIDSeed("img_abc", 42), saltedIDSeed("img_abc", 42); got != want {
			t.Fatalf("saltedIDSeed not stable: %d then %d", want, got)
		}
	}
}

func TestSaltChangesTheDraw(t *testing.T) {
	differs := 0
	for i := range 500 {
		if saltedIndexSeed("W", i, 1) != saltedIndexSeed("W", i, 2) {
			differs++
		}
	}
	// A salt that only perturbed a few indices would leave most photos identical,
	// so two runs would look like one and the flag would be pointless. A hash
	// should separate effectively all of them.
	if differs < 450 {
		t.Errorf("only %d/500 draws differ between two salts — the salt is barely reaching the draw", differs)
	}

	if saltedBurstSeed(3, 2, 1) == saltedBurstSeed(3, 2, 2) {
		t.Error("saltedBurstSeed ignores the salt")
	}
	if saltedIDSeed("img_abc", 1) == saltedIDSeed("img_abc", 2) {
		t.Error("saltedIDSeed ignores the salt")
	}
}

func TestSaltZeroIsTheUnsaltedBehaviour(t *testing.T) {
	// Zero means "no salt", not "salt of zero". If it hashed as a real salt, every
	// existing run's photos would move the first time --seed was used, and the
	// idempotency the loader tests rely on would quietly stop holding for anyone
	// who set the flag to 0.
	for i := range 200 {
		if got, want := saltedIndexSeed("W", i, 0), indexSeed("W", i); got != want {
			t.Fatalf("saltedIndexSeed(\"W\", %d, 0) = %d, want the unsalted %d", i, got, want)
		}
		if got, want := saltedBurstSeed(i, 1, 0), burstSeed(i, 1); got != want {
			t.Fatalf("saltedBurstSeed(%d, 1, 0) = %d, want the unsalted %d", i, got, want)
		}
	}
	if got, want := saltedIDSeed("img_abc", 0), idSeed("img_abc"); got != want {
		t.Errorf("saltedIDSeed(\"img_abc\", 0) = %d, want the unsalted %d", got, want)
	}
}

func TestSaltOf(t *testing.T) {
	if got := saltOf(nil); got != 0 {
		t.Errorf("saltOf(nil) = %d, want 0", got)
	}
	if got := saltOf([]int64{}); got != 0 {
		t.Errorf("saltOf([]) = %d, want 0", got)
	}
	if got := saltOf([]int64{42}); got != 42 {
		t.Errorf("saltOf([42]) = %d, want 42", got)
	}
}
