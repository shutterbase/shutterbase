package seed

import (
	"fmt"
	"testing"
)

// indexSeed must mix the prefix's BYTES, not its length. The old term was
// index*1_000_003 + len(prefix), so ANY two prefixes of equal length produced
// the literally identical stream for every index — "LW" and "WW" (or a future
// "W" retitled "W2" and a new "W2") would be the same draw, which is exactly
// the independence between the two load seeders that the function's comment
// claims. ("W" and "LW" happen to differ by length and were safe; the hazard is
// the whole equal-length class, and the comment asserted a property the code did
// not have.)
func TestIndexSeedMixesPrefixBytes(t *testing.T) {
	for i := range 5000 {
		if got, want := indexSeed("LW", i), indexSeed("WW", i); got == want {
			t.Fatalf("index %d: equal-length prefixes \"LW\" and \"WW\" share a stream (%d)", i, got)
		}
		if got, want := indexSeed("W", i), indexSeed("LWL", i); got == want {
			t.Fatalf("index %d: equal-length prefixes \"W\" and \"LWL\" share a stream (%d)", i, got)
		}
		// Distinct lengths must stay distinct too, or the two load seeders'
		// streams would drift into each other.
		if got, want := indexSeed("W", i), indexSeed("LW", i); got == want {
			t.Fatalf("index %d: \"W\" and \"LW\" share a stream (%d)", i, got)
		}
	}
	// …and it must still be a function of the index, not merely of the prefix.
	seen := map[int64]int{}
	for i := range 5000 {
		seen[indexSeed("W", i)]++
	}
	for i, n := range seen {
		if n > 1 {
			t.Fatalf("indexSeed(\"W\") collided at %d: %d indices share the seed", i, n)
		}
	}
}

// The image id is 15 chars for EVERY row (StringIDMixin MaxLen(15)), which is
// why the per-image stream was once seeded from len(id) and handed the whole
// project one tag set. The VALUE has to be what varies.
func TestIDSeedVariesWithTheIDValue(t *testing.T) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	first := make([]byte, 15)
	for i := range first {
		first[i] = alphabet[i]
	}
	other := append([]byte(nil), first...)
	other[14]++ // last char only

	if idSeed(string(first)) == idSeed(string(other)) {
		t.Fatal("two different ids produced the same seed")
	}
	if got := idSeed(string(first)); got != idSeed(string(first)) {
		t.Fatalf("idSeed is not deterministic: %d then %d", got, idSeed(string(first)))
	}
	// Same length, different value: the exact case len() could not tell apart.
	if len(first) != len(other) {
		t.Fatalf("probe ids differ in length, the test proves nothing")
	}
}

// burstSeed packs (burst, slot) into one word. If the two indices were folded
// into a single int instead, a high slot count in one burst would collide with a
// later burst's low slots and two different photos would share a stream.
func TestBurstSeedDoesNotSpillAcrossBursts(t *testing.T) {
	seen := map[int64]string{}
	for b := 0; b < 35; b++ {
		for slot := 0; slot < 2000; slot++ {
			got := burstSeed(b, slot)
			key := fmt.Sprintf("%d/%d", b, slot)
			if prev, dup := seen[got]; dup {
				t.Fatalf("burstSeed collides: %s and %s both hash to %d", prev, key, got)
			}
			seen[got] = key
		}
	}
}

// photoExtras draws the count AND the tags off ONE stream. Two rngs seeded with
// the same value each consume the same first Intn(10), which pinned a 1-extra
// photo's only tag to 3 of the 10 pool entries and a 3-extra photo's to 2. Over
// 20000 indices every pool entry must now be reachable at every count.
func TestPhotoExtrasDrawsCountAndTagsFromOneStream(t *testing.T) {
	pool := make([]string, 10)
	for i := range pool {
		pool[i] = fmt.Sprintf("Tag%02d", i)
	}
	buckets := map[int]map[string]int{}
	counts := map[int]int{}
	for i := range 20000 {
		tags := photoExtras(rngFor(indexSeed("W", i)), pool)
		counts[len(tags)]++
		if buckets[len(tags)] == nil {
			buckets[len(tags)] = map[string]int{}
		}
		for _, tag := range tags {
			buckets[len(tags)][tag]++
		}
	}
	// The documented 30/50/20 split, with room to spare.
	for n, floor := range map[int]int{1: 5000, 2: 9000, 3: 3500} {
		if counts[n] < floor {
			t.Errorf("%d photos carry %d extra tags, want >%d", counts[n], n, floor)
		}
	}
	for n := 1; n <= 3; n++ {
		if len(buckets[n]) != len(pool) {
			t.Errorf("only %d of %d pool tags are reachable with %d extra tags: %v",
				len(buckets[n]), len(pool), n, buckets[n])
		}
	}
}
