package seed_test

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/seed"
)

func assignmentCount(t *testing.T, c *ent.Client, imageID string) int {
	t.Helper()
	n, err := c.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(imageID)).
		Count(context.Background())
	require.NoError(t, err)
	return n
}

// drawPool resolves the ids the loaders draw their random extras from.
//
// The pool GREW: it used to be Tag00–Tag09 alone, and now holds every generated
// team tag as well, because the loaders create those tags for the facets and then
// never assigned them — 80 tags on zero photos, which is worse than useless
// because repository.GetImageTagFacets drops zero-count tags, so they showed up in
// no facet at all.
//
// Spelled out from the generator rather than swept out of m.Tags: the manifest also
// holds Default, the internal marker and the $DATE/$WEEKDAY templates, none of
// which the draw can pick, and a pool silently padded with them would make the
// per-bucket coverage checks below pass without the draw having reached the team
// tags.
func drawPool(t *testing.T, m *seed.Manifest) []string {
	t.Helper()
	generated := seed.GeneratedTeamTags()
	pool := make([]string, 0, len(generated)+10)
	for _, tag := range generated {
		id := m.Tags[tag.Name]
		require.NotEmpty(t, id, "generated team tag %q is not on the manifest", tag.Name)
		pool = append(pool, id)
	}
	for n := 0; n < 10; n++ {
		id := m.Tags[fmt.Sprintf("Tag%02d", n)]
		require.NotEmpty(t, id, "Tag%02d must exist", n)
		pool = append(pool, id)
	}
	return pool
}

func loadPhotos(t *testing.T, c *ent.Client, prefix string) []*ent.Image {
	t.Helper()
	rows, err := c.Image.Query().Where(image.ComputedFileNameHasPrefix(prefix)).All(context.Background())
	require.NoError(t, err)
	return rows
}

// The load seeders back the time-range slider's density strip, so these
// properties are load-bearing and each one was broken at some point:
//
//   - exact counts (`--week N` / `--last-week N` must seed N photos — plain
//     per-burst truncation lost the remainder and starved the newest day),
//   - no photo dated in the future (photos spread forward from referenceNow
//     read as future captures to EXIF export, slideshows and recency order),
//   - an idempotent re-run that adds no duplicate (image, tag) assignments,
//   - the denormalized images.imageTags jsonb agreeing with the assignment
//     rows. The gallery filter (repository.buildImagePredicates uses
//     sqljson.ValueContains) and ToImageResponse read that jsonb, NEVER the
//     assignment rows — a seeder that writes assignments without rebuilding it
//     produces tags the app cannot see.
func TestLoadSeedersHonourCountsAndDates(t *testing.T) {
	// 500 per seeder: the per-bucket tag-coverage check below needs ~150 photos
	// per extra-count bucket before a legitimately uniform draw is safe from
	// flaking.
	const perSeeder = 500

	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)
	now := time.Now()

	require.NoError(t, seed.SeedWeekOfPhotos(ctx, c, m, seed.SevenDaysEndingAt(now), perSeeder))
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, c, m, seed.SevenDaysEndingAt(now), perSeeder))

	week := loadPhotos(t, c, "FSG_W")
	lastWeek := loadPhotos(t, c, "FSG_LW")
	// require, not assert: every count-derived assertion below (the 30/50/20
	// split, the per-bucket tag coverage) is only meaningful on the exact
	// requested population, and a short set cascades into a wall of unrelated
	// failures instead of stopping at the count regression.
	require.Len(t, week, perSeeder, "--week N must seed exactly N photos")
	require.Len(t, lastWeek, perSeeder, "--last-week N must seed exactly N photos")

	all := make([]*ent.Image, 0, len(week)+len(lastWeek))
	all = append(all, week...)
	all = append(all, lastWeek...)
	for _, img := range all {
		require.NotNil(t, img.CapturedAtCorrected)
		assert.False(t, img.CapturedAtCorrected.After(now), "load-seeded photos must not be dated in the future")
	}

	// The 30/50/20 split over 1/2/3 extra tags is part of the documented
	// contract, and "distinct" matters: drawing with replacement used to hand one
	// photo the same tag twice and the duplicate insert was discarded as a
	// constraint error, so the photo silently ended up with fewer tags.
	defaultTag := m.Tags["Default"]
	extraByCount := map[int]int{}
	pool := drawPool(t, m)
	for _, img := range all {
		require.NotEmpty(t, img.ImageTags, "imageTags jsonb must be populated")
		assert.Contains(t, img.ImageTags, defaultTag, "Default must be in the jsonb read-model")
		assert.ElementsMatch(t, img.ImageTags, uniqueSorted(img.ImageTags), "jsonb must not contain duplicates")

		// the jsonb read-model and the assignment rows must agree exactly
		assert.ElementsMatch(t, uniqueSorted(assignmentTags(t, c, img.ID)), uniqueSorted(img.ImageTags),
			"imageTags jsonb must match the assignment rows for %s", img.ComputedFileName)

		// Every photo also carries two unconditional tags: its capture date
		// (20261002) and its weekday (Thursday). They are derived from the instant,
		// not drawn, so they do not count against the 1-3 random pool — hence 5 as
		// the ceiling: Default + 3 random + date + weekday.
		extra := len(img.ImageTags) - 1 // minus Default
		assert.GreaterOrEqual(t, extra, 3, "Default + at least one random extra + date + weekday")
		assert.LessOrEqual(t, extra, 5, "%s: Default + at most three random extras + date + weekday, got %v",
			img.ComputedFileName, tagNames(t, c, img.ID))

		names := tagNames(t, c, img.ID)
		assert.Contains(t, names, seed.DayTagName(*img.CapturedAtCorrected),
			"%s must carry its capture date", img.ComputedFileName)
		assert.Contains(t, names, seed.WeekdayTagName(*img.CapturedAtCorrected),
			"%s must carry its capture weekday", img.ComputedFileName)

		// The 30/50/20 split is about the RANDOM pool only. The two calendar tags
		// are derived from the instant and are present on every photo, so counting
		// them here would shift every bucket by two and make the split unfalsifiable.
		random := 0
		for _, id := range img.ImageTags {
			if slices.Contains(pool, id) {
				random++
			}
		}
		assert.GreaterOrEqual(t, random, 1, "%s: at least one random extra tag", img.ComputedFileName)
		assert.LessOrEqual(t, random, 3, "%s: at most three random extra tags, got %v",
			img.ComputedFileName, names)
		extraByCount[random]++
	}
	// Over 1000 photos the documented 30/50/20 split is unambiguous. A constant
	// draw (always 1 extra tag) would put everything in extraByCount[1].
	assert.Greater(t, extraByCount[1], 200, "some photos carry exactly 1 extra tag")
	assert.Greater(t, extraByCount[2], 400, "some photos carry exactly 2 extra tags")
	assert.Greater(t, extraByCount[3], 100, "some photos carry exactly 3 extra tags")

	// …and WHICH tags matters just as much as how many, checked PER SEEDER. The
	// count and the first tag used to be drawn from two rngs seeded with the same
	// value, so both consumed the same first Intn(10) of a 10-element pool: a
	// 1-extra photo's single tag was confined to 3 of the 10 entries and a
	// 3-extra photo's to 2. Seven pool tags were unreachable for those photos.
	// Merging both seeders into one bucket map would hide exactly that — the
	// healthy seeder covers the dead one's gaps.
	for _, group := range []struct {
		name  string
		imgs  []*ent.Image
		share string
	}{
		{"FSG_W", week, "week"},
		{"FSG_LW", lastWeek, "last-week"},
	} {
		bucketsUsed := map[int]map[string]int{}
		counts := map[int]int{}
		for _, img := range group.imgs {
			// Pool tags only. The two calendar tags ride along on every photo, so
			// counting by len(ImageTags) would shift every bucket by two and this
			// whole uniformity check would look at counts that never occur.
			// Two passes on purpose: the bucket key is the photo's FINAL
			// count, so crediting inside the counting loop would file a 2-tag
			// photo's first tag under bucket 1 and hide exactly the skew this
			// check exists to catch.
			poolTags := make([]string, 0, 3)
			for _, tagID := range img.ImageTags {
				if slices.Contains(pool, tagID) {
					poolTags = append(poolTags, tagID)
				}
			}
			extra := len(poolTags)
			if bucketsUsed[extra] == nil {
				bucketsUsed[extra] = map[string]int{}
			}
			for _, tagID := range poolTags {
				bucketsUsed[extra][tagID]++
			}
			counts[extra]++
		}
		// The pool GREW from 10 ids to 90, and "every pool tag in every bucket" is no
		// longer an affordable assertion at this population: reaching all 90 from a
		// bucket of ~150 single-tag photos is a coupon-collector problem, so ~17 tags
		// per bucket would be unseen on any given run and the check would be a coin
		// flip rather than a check. Two assertions replace it, neither of which the
		// old pool satisfied by accident:
		//
		//   - each bucket still draws WIDELY. The defect being caught is a bucket
		//     confined to two or three pool entries (the 3-of-10 / 2-of-10 skew above),
		//     so a floor at half the pool is four standard deviations below what the
		//     SMALLEST bucket reaches and still fails loudly on it.
		//   - every pool tag reaches SOME photo in this seeder. That is the property
		//     the skew broke — vocabulary the draw can never produce — and 500 photos
		//     (~1000 draws) clears a 90-entry pool with room to spare.
		floor := len(pool) / 2
		used := map[string]bool{}
		for n := 1; n <= 3; n++ {
			require.Positive(t, counts[n], "%s: no photos with %d extra tags", group.name, n)
			assert.GreaterOrEqual(t, len(bucketsUsed[n]), floor,
				"%s: a photo with %d extra tag(s) drew from only %d of the %d pool tags — the draw is not uniform over the pool",
				group.share, n, len(bucketsUsed[n]), len(pool))
			for _, tagID := range pool {
				if bucketsUsed[n][tagID] > 0 {
					used[tagID] = true
				}
			}
		}
		for _, tagID := range pool {
			assert.True(t, used[tagID],
				"%s: tag %s is on no photo at all — it is in the draw pool and the draw cannot reach it",
				group.share, tagID)
		}
	}

	// Re-run: same photo count, and no duplicate assignments anywhere.
	countsBefore := make(map[string]int, len(week))
	for _, img := range week {
		countsBefore[img.ID] = assignmentCount(t, c, img.ID)
	}
	require.NoError(t, seed.SeedWeekOfPhotos(ctx, c, m, seed.SevenDaysEndingAt(now), perSeeder))
	week2 := loadPhotos(t, c, "FSG_W")
	require.Len(t, week2, perSeeder, "re-run stays idempotent")
	for _, img := range week2 {
		if before, ok := countsBefore[img.ID]; ok {
			assert.Equal(t, before, assignmentCount(t, c, img.ID), "re-run adds no duplicate assignments")
		}
	}

	// A re-run at a DIFFERENT wall clock must be a no-op too. The tag draw used
	// to be seeded from referenceNow, and cmd/seed passes a fresh time.Now() on
	// every run, so each re-run picked a different set and kept appending tags
	// until every photo carried all ten.
	require.NoError(t, seed.SeedWeekOfPhotos(ctx, c, m, seed.SevenDaysEndingAt(now.Add(72*time.Hour)), perSeeder))
	after := loadPhotos(t, c, "FSG_W")
	require.Len(t, after, perSeeder, "a later re-run adds no photos")
	for _, img := range after {
		// Default + 3 random + date + weekday. The ceiling matters more than the
		// count: a re-run that re-drew would keep appending until every pool tag
		// was present, and only an upper bound catches that.
		assert.LessOrEqual(t, len(img.ImageTags), 6,
			"re-running at a different wall clock must not append tags to %s", img.ComputedFileName)
	}

	// The same guard for the last-week seeder, which had the identical bug: its
	// extras came off the single wall-clock-seeded rng, so every re-run at a new
	// time appended up to 3 more tags per photo until all ten were present. It
	// was invisible because only the week seeder was ever re-run here.
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, c, m, seed.SevenDaysEndingAt(now.Add(72*time.Hour)), perSeeder))
	afterLW := loadPhotos(t, c, "FSG_LW")
	require.Len(t, afterLW, perSeeder, "a later last-week re-run adds no photos")
	for _, img := range afterLW {
		assert.LessOrEqual(t, len(img.ImageTags), 6,
			"re-running the last-week seeder at a different wall clock must not append tags to %s", img.ComputedFileName)
	}
}

// The last-week seeder draws its whole burst layout AND its tag sets. Both used
// to come off one rng seeded from referenceNow.UnixNano(), so the same
// `--last-week N` at two different moments produced two different timelines: a
// top-up was not a superset of a single larger run, and each re-run appended
// tags. The invariant is that a photo's instant and its tag set depend only on
// its index and the window.
//
// UTC reference instants keep the comparison honest: weekStart is derived with
// AddDate, so a reference pair straddling a DST transition in the host's zone
// would shift whole days by an hour and the instants would legitimately differ.
func TestLastWeekLayoutIgnoresWallClock(t *testing.T) {
	ctx := context.Background()
	refA := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	refB := refA.Add(72 * time.Hour)

	cA := sqliteClient(t)
	mA, err := seed.Seed(ctx, cA, refA)
	require.NoError(t, err)
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, cA, mA, seed.SevenDaysEndingAt(refA), 300))

	cB := sqliteClient(t)
	mB, err := seed.Seed(ctx, cB, refB)
	require.NoError(t, err)
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, cB, mB, seed.SevenDaysEndingAt(refB), 300))

	byNameA := randomTagSetsByName(t, cA, mA, "FSG_LW")
	byNameB := randomTagSetsByName(t, cB, mB, "FSG_LW")
	require.Len(t, byNameA, 300)
	require.Len(t, byNameB, 300)
	for name, tagsA := range byNameA {
		assert.Equal(t, tagsA, byNameB[name],
			"a photo's tag set must not depend on the wall clock: %s", name)
	}

	// The calendar tags are the deliberate exception: they are derived from the
	// capture instant, so two windows 72h apart MUST disagree on them. Asserting
	// that here is the point — a loader that resolved them from the clock instead
	// of the instant would put both runs' photos in the same day bucket.
	for name, tagsA := range tagNameSetsByName(t, cA, mA, "FSG_LW") {
		if a, b := tagsA, tagNameSetsByName(t, cB, mB, "FSG_LW")[name]; slices.Equal(a, b) {
			t.Errorf("%s: two windows 72h apart produced identical tag sets; the calendar "+
				"tags are not tracking the capture instant", name)
		}
	}

	// Instants: compare each photo's position WITHIN the seeded window, so the
	// two 7-day windows 72h apart line up.
	relA := relativeToWeekStart(t, loadPhotos(t, cA, "FSG_LW"), refA)
	relB := relativeToWeekStart(t, loadPhotos(t, cB, "FSG_LW"), refB)
	require.Len(t, relA, 300)
	for name, offsetA := range relA {
		require.Contains(t, relB, name)
		assert.Equal(t, offsetA, relB[name],
			"a photo's instant within the window must not depend on the wall clock: %s", name)
	}
}

func relativeToWeekStart(t *testing.T, imgs []*ent.Image, referenceNow time.Time) map[string]time.Duration {
	t.Helper()
	weekStart := referenceNow.AddDate(0, 0, -7)
	out := make(map[string]time.Duration, len(imgs))
	for _, img := range imgs {
		require.NotNil(t, img.CapturedAtCorrected, "%s has no corrected capture time", img.ComputedFileName)
		out[img.ComputedFileName] = img.CapturedAtCorrected.Sub(weekStart)
	}
	return out
}

// The load seeders denormalize the Default tag into every photo's jsonb and into
// every assignment row. A manifest merged from a hand-edited file — or a project
// whose Default tag was deleted — resolves to the empty string, and both writes
// then carry "" as a foreign key: an opaque constraint error that aborts the
// whole 500-row chunk. The empty case has to be reported by name instead.
func TestLoadSeedersRejectAnEmptyDefaultTag(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)
	now := time.Now()
	delete(m.Tags, "Default")

	errWeek := seed.SeedWeekOfPhotos(ctx, c, m, seed.SevenDaysEndingAt(now), 20)
	require.Error(t, errWeek, "a manifest with no Default tag must fail loudly, not write an empty-string FK")
	assert.Contains(t, errWeek.Error(), "Default", "the error must name the missing tag")

	errLastWeek := seed.SeedLastWeekPhotos(ctx, c, m, seed.SevenDaysEndingAt(now), 20)
	require.Error(t, errLastWeek, "a manifest with no Default tag must fail loudly, not write an empty-string FK")
	assert.Contains(t, errLastWeek.Error(), "Default", "the error must name the missing tag")

	// Nothing may be half-written: the images are created inside the same tx as
	// their assignments, so the failed chunk left nothing behind.
	assert.Empty(t, loadPhotos(t, c, "FSG_W"), "the failed week chunk must not leave photos behind")
	assert.Empty(t, loadPhotos(t, c, "FSG_LW"), "the failed last-week chunk must not leave photos behind")
}

// Growing the counts exercises the paths a single-chunk run never reaches: the
// partial-chunk boundary in SeedLastWeekPhotos, where `created` is compacted
// (existing names are skipped) while the extras slice is not. Indexing them by
// position gave 298 of 300 new photos another photo's tag set.
//
// The jsonb-vs-assignment-rows check CANNOT see that: both are written from the
// same (wrong) value, so they stay perfectly self-consistent. The invariant that
// does catch it is that a photo's tag set is a function of its INDEX alone — it
// does not depend on how many photos existed when the seeder ran. So: seed 700
// in one shot on one database, seed 200-then-700 on another, and require all 700
// photos — the 200 the top-up inherited included — to carry identical tag sets.
func TestLoadSeedersGrowAcrossChunkBoundaries(t *testing.T) {
	ctx := context.Background()

	// A: one shot.
	cA := sqliteClient(t)
	mA, err := seed.Seed(ctx, cA, time.Now())
	require.NoError(t, err)
	nowA := time.Now()
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, cA, mA, seed.SevenDaysEndingAt(nowA), 700))

	// B: 200 first, then top up to 700 — the second run's first chunk is
	// half-existing, which is where the compaction bug lives.
	cB := sqliteClient(t)
	mB, err := seed.Seed(ctx, cB, time.Now())
	require.NoError(t, err)
	nowB := nowA
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, cB, mB, seed.SevenDaysEndingAt(nowB), 200))
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, cB, mB, seed.SevenDaysEndingAt(nowB), 700))

	require.Len(t, loadPhotos(t, cB, "FSG_LW"), 700, "growing the count must top the set up, not duplicate it")

	// Compare by tag NAME, not id: each seed run mints fresh tag ids, so the raw
	// id sets of two independent databases never match.
	//
	// EVERY photo is compared, including the 200 run B created in its first
	// pass. The skip that used to sit here (`idx < 200`) excused exactly the
	// overlapping photos the test exists to compare.
	// The RANDOM tags only. The calendar tags are excluded because the burst
	// layout is an apportionment of the count: cB seeded its first 200 photos
	// during the 200-run, on a 200-photo layout, and a 700-run never moves an
	// existing photo. So cA and cB legitimately disagree on when those photos were
	// taken, and a full-set comparison here would be asserting a property the
	// loaders never had.
	byNameA := randomTagSetsByName(t, cA, mA, "FSG_LW")
	byNameB := randomTagSetsByName(t, cB, mB, "FSG_LW")
	require.Len(t, byNameA, 700)
	require.Len(t, byNameB, 700)

	mismatched := 0
	for name, tagsA := range byNameA {
		if !slices.Equal(tagsA, byNameB[name]) {
			mismatched++
		}
	}
	assert.Zero(t, mismatched,
		"a photo's tag set must depend only on its index, not on how many photos existed when the seeder ran")

	// And the jsonb read-model must still agree with the assignment rows.
	for _, img := range loadPhotos(t, cB, "FSG_LW") {
		assert.ElementsMatch(t, uniqueSorted(assignmentTags(t, cB, img.ID)), uniqueSorted(img.ImageTags),
			"imageTags jsonb must match the assignment rows for %s", img.ComputedFileName)
	}

	// Re-run at the larger count: nothing new, nothing duplicated.
	require.NoError(t, seed.SeedLastWeekPhotos(ctx, cB, mB, seed.SevenDaysEndingAt(nowB), 700))
	assert.Len(t, loadPhotos(t, cB, "FSG_LW"), 700, "last-week re-run stays idempotent")
}

// TagExistingPhotos adds assignment rows to images that ALREADY have tags, so
// it is the only path where the denormalized jsonb can drift: the image is not
// recreated, and nothing else would refresh images.imageTags. The gallery filter
// (repository.buildImagePredicates uses sqljson.ValueContains) and
// ToImageResponse read that jsonb, never the assignment rows — so without the
// rebuild, --tag-existing adds tags the app cannot see.
func TestTagExistingPhotosRebuildsTheJSONBReadModel(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)

	before, err := c.Image.Query().Where(image.ProjectID(m.Project)).All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, before)

	require.NoError(t, seed.TagExistingPhotos(ctx, c, m, time.Now(), cleanupOffset()))

	after, err := c.Image.Query().Where(image.ProjectID(m.Project)).All(ctx)
	require.NoError(t, err)
	require.Len(t, after, len(before))

	grew := 0
	for _, img := range after {
		rowTags := uniqueSorted(assignmentTags(t, c, img.ID))
		// the jsonb read-model and the assignment rows must agree exactly — the
		// whole point: without the rebuild they drift and the app cannot see the
		// new tags
		assert.ElementsMatch(t, rowTags, uniqueSorted(img.ImageTags),
			"imageTags jsonb must match the assignment rows for %s", img.ComputedFileName)
		// Matched by name prefix, like the loader does. main has no
		// SeedTimeRangeCluster, so there is no manifest field to read; and the
		// prefix is what the loader itself keys on, so the two cannot drift.
		if strings.HasPrefix(img.ComputedFileName, seed.TimeRangeClusterPrefix) {
			// the midnight cluster is the time-range filter's UNTAGGED control
			assert.Empty(t, img.ImageTags, "the midnight cluster must stay untagged")
			continue
		}
		assert.Contains(t, img.ImageTags, m.Tags["Default"], "Default stays in the jsonb")
		old := findImage(before, img.ID)
		if old != nil && len(img.ImageTags) > len(old.ImageTags) {
			grew++
		}
	}
	assert.Greater(t, grew, 0, "TagExistingPhotos must actually add tags, or this proves nothing")
}

// Every photo in the project must get its OWN tag set. The per-image stream was
// seeded from len(img.ID), and StringIDMixin is field.String("id").MaxLen(15) —
// so that value is 15 for EVERY image and the whole project shared one stream:
// the documented 30/50/20 split collapsed and eight of the ten tags sat on zero
// photos while the tenth landed on all of them. Seeding from the id VALUE fixes
// it. This needs a population: the base fixtures are only 3 photos, and 3 draws
// collide by chance too often for a test.
func TestTagExistingPhotosVariesTagsPerImage(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)

	const bare = 400
	bareIDs := insertBareImages(t, c, m, bare)

	require.NoError(t, seed.TagExistingPhotos(ctx, c, m, time.Now(), cleanupOffset()))

	distinct := map[string]int{}
	used := map[string]int{}
	for _, id := range bareIDs {
		img, err := c.Image.Get(ctx, id)
		require.NoError(t, err)
		tags := uniqueSorted(img.ImageTags)
		require.Len(t, tags, assignmentCount(t, c, id), "jsonb must match the assignment rows for %s", img.ComputedFileName)
		key := strings.Join(tags, ",")
		distinct[key]++
		for _, tagID := range tags {
			used[tagID]++
		}
	}
	// 400 photos over 175 possible (count, set) combinations: the expected number
	// of distinct sets is ~157, so >50 has no flakiness, while the seeded-on-
	// length bug produces exactly 1.
	assert.Greater(t, len(distinct), 50,
		"TagExistingPhotos must draw a different tag set per image; got %d distinct sets over %d photos", len(distinct), bare)
	for n := 0; n < 10; n++ {
		tagID := m.Tags[fmt.Sprintf("Tag%02d", n)]
		require.NotEmpty(t, tagID, "Tag%02d must exist", n)
		assert.Positive(t, used[tagID], "Tag%02d is on zero photos", n)
	}
}

// insertBareImages adds `n` photos to the seeded project carrying no tag
// assignments and no jsonb, so TagExistingPhotos is measured on images whose
// whole tag set comes from its own draw. Bulk-inserted in batches: SQLite caps
// the number of bound variables per statement, so one CreateBulk over a few
// thousand images fails outright.
func insertBareImages(t *testing.T, c *ent.Client, m *seed.Manifest, n int) []string {
	t.Helper()
	const batch = 500
	ids := make([]string, 0, n)
	for start := 0; start < n; start += batch {
		build := make([]*ent.ImageCreate, 0, min(batch, n-start))
		for i := start; i < min(start+batch, n); i++ {
			build = append(build, c.Image.Create().
				SetFileName(fmt.Sprintf("BARE_%05d.jpg", i)).
				SetComputedFileName(fmt.Sprintf("FSG_B%05d.jpg", i)).
				SetStorageId(fmt.Sprintf("seedbare%08d", i)).
				SetSize(1024).
				SetWidth(6000).
				SetHeight(4000).
				SetCapturedAt(time.Now()).
				SetCapturedAtCorrected(time.Now()).
				SetUserID(m.Users["projectEditor"]).
				SetUploadID(m.Upload).
				SetProjectID(m.Project).
				SetCameraID(m.Cameras["fresh"]))
		}
		rows, err := c.Image.CreateBulk(build...).Save(context.Background())
		require.NoError(t, err)
		require.Len(t, rows, len(build))
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
	}
	require.Len(t, ids, n)
	return ids
}

func findImage(in []*ent.Image, id string) *ent.Image {
	for _, i := range in {
		if i.ID == id {
			return i
		}
	}
	return nil
}

// tagNameSetsByName maps each photo's jsonb read-model to sorted tag NAMES, so
// two independently seeded databases can be compared.
func tagNameSetsByName(t *testing.T, c *ent.Client, m *seed.Manifest, prefix string) map[string][]string {
	t.Helper()
	byID := make(map[string]string, len(m.Tags))
	for name, id := range m.Tags {
		byID[id] = name
	}
	out := map[string][]string{}
	for _, img := range loadPhotos(t, c, prefix) {
		names := make([]string, 0, len(img.ImageTags))
		for _, id := range img.ImageTags {
			name, ok := byID[id]
			require.True(t, ok, "tag %s on %s is not in the manifest", id, img.ComputedFileName)
			names = append(names, name)
		}
		out[img.ComputedFileName] = uniqueSorted(names)
	}
	return out
}

func assignmentTags(t *testing.T, c *ent.Client, imageID string) []string {
	t.Helper()
	rows, err := c.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(imageID)).
		All(context.Background())
	require.NoError(t, err)
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ImageTagID)
	}
	return out
}

// randomTagSetsByName is tagNameSetsByName with the calendar tags removed, so two
// runs whose windows sit on different dates can still be compared on the part that
// must be identical: the drawn pool.
func randomTagSetsByName(t *testing.T, c *ent.Client, m *seed.Manifest, prefix string) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for name, tags := range tagNameSetsByName(t, c, m, prefix) {
		random := make([]string, 0, len(tags))
		for _, tag := range tags {
			if !seed.CalendarTagPrefix(tag) {
				random = append(random, tag)
			}
		}
		out[name] = uniqueSorted(random)
	}
	return out
}

// tagNames resolves an image's assignment rows to tag NAMES, so an assertion can
// read "Thursday" rather than an opaque id.
func tagNames(t *testing.T, c *ent.Client, imageID string) []string {
	t.Helper()
	ctx := context.Background()
	ids := assignmentTags(t, c, imageID)
	if len(ids) == 0 {
		return nil
	}
	rows, err := c.ImageTag.Query().Where(imagetag.IDIn(ids...)).All(ctx)
	require.NoError(t, err)
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Name)
	}
	return out
}

// Fewer photos than bursts (35) is the degenerate case the largest-remainder
// pass used to mishandle: every burst floors to one photo, the fix-up pass
// could not decrement, and the imgIdx cap then took every photo from the
// OLDEST day — the exact starvation the apportionment exists to prevent.
func TestLoadSeedersBelowBurstCountStillFill(t *testing.T) {
	ctx := context.Background()
	c := sqliteClient(t)
	m, err := seed.Seed(ctx, c, time.Now())
	require.NoError(t, err)
	now := time.Now()

	require.NoError(t, seed.SeedLastWeekPhotos(ctx, c, m, seed.SevenDaysEndingAt(now), 12))
	lw := loadPhotos(t, c, "FSG_LW")
	require.Len(t, lw, 12, "a count below the burst count still seeds exactly that many")

	// The seeded window must not be a single day: 12 photos all from day 0 is
	// the starvation signature.
	days := map[string]struct{}{}
	for _, img := range lw {
		require.NotNil(t, img.CapturedAtCorrected)
		days[img.CapturedAtCorrected.Format("2006-01-02")] = struct{}{}
	}
	assert.Greater(t, len(days), 1, "photos must spread across days, not pile into the oldest")
}

// uniqueSorted returns a sorted, de-duplicated copy — comparing tag sets without
// depending on assignment order.
func uniqueSorted(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return slices.Compact(out)
}
