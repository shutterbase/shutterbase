package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/seed"
)

func TestGetImagePosition(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	params := func() *repository.GetImageParameters {
		return &repository.GetImageParameters{
			ProjectID:            m.Project,
			PaginationParameters: &repository.PaginationParameters{Sort: "capturedAtCorrected", Order: "desc"},
		}
	}

	// the position must match the index the list query itself would yield
	items, total, err := repo.GetImages(ctx, params())
	require.NoError(t, err)
	require.Equal(t, total, len(items), "seed fits in one page")
	require.GreaterOrEqual(t, len(items), 2, "seed provides multiple images")

	for i, img := range items {
		pos, err := repo.GetImagePosition(ctx, params(), img.ID, 2000)
		require.NoError(t, err)
		assert.Equal(t, i, pos, "image %s", img.ID)
	}

	// flipping the order flips the position
	asc := params()
	asc.PaginationParameters.Order = "asc"
	pos, err := repo.GetImagePosition(ctx, asc, items[0].ID, 2000)
	require.NoError(t, err)
	assert.Equal(t, len(items)-1, pos)

	// unknown id and out-of-scan-window both answer -1
	pos, err = repo.GetImagePosition(ctx, params(), "no-such-image", 2000)
	require.NoError(t, err)
	assert.Equal(t, -1, pos)

	pos, err = repo.GetImagePosition(ctx, params(), items[len(items)-1].ID, 1)
	require.NoError(t, err)
	assert.Equal(t, -1, pos)

	// a filter that excludes the image answers -1, not an error
	filtered := params()
	otherProject := "no-such-project"
	filtered.ProjectID = otherProject
	pos, err = repo.GetImagePosition(ctx, filtered, items[0].ID, 2000)
	require.NoError(t, err)
	assert.Equal(t, -1, pos)
}

// Time-range filtering on capturedAtCorrected: inclusive bounds, open-ended
// single sides, NULL-corrected exclusion, and combination with the tag
// filters. Rides the seed's midnight cluster (23:55→00:10 event-local,
// yesterday) whose first/last photos sit exactly on the boundary instants;
// the three base photos are ~referenceNow, i.e. AFTER the whole cluster.
func TestGetImagesTimeRange(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	paged := &repository.PaginationParameters{Sort: "capturedAtCorrected", Order: "asc"}
	params := func() *repository.GetImageParameters {
		return &repository.GetImageParameters{ProjectID: m.Project, PaginationParameters: paged}
	}

	// an image without a corrected capture time: matches unbounded queries only
	_, err := repo.Client.Image.Create().
		SetFileName("N_0001.jpg").SetComputedFileName("FSG_9999.jpg").
		SetStorageId("seednr00000001").SetSize(1).
		SetUserID(m.Users["projectEditor"]).SetUploadID(m.Upload).
		SetProjectID(m.Project).SetCameraID(m.Cameras["fresh"]).
		Save(ctx)
	require.NoError(t, err)

	start, end := m.TimeRangeStart, m.TimeRangeEnd

	// no bounds: everything, including the NULL-corrected image
	_, total, err := repo.GetImages(ctx, params())
	require.NoError(t, err)
	assert.Equal(t, 12, total, "3 base + 8 midnight + 1 uncorrected")

	// closed range [start, end]: the eight cluster photos, boundary-equal ones included
	from, to := start, end
	bounded := params()
	bounded.FromCapturedAtCorrected = &from
	bounded.ToCapturedAtCorrected = &to
	items, total, err := repo.GetImages(ctx, bounded)
	require.NoError(t, err)
	assert.Equal(t, 8, total)
	assert.Equal(t, m.TimeRangeImages[0], items[0].ID, "==from is inclusive")
	assert.Equal(t, m.TimeRangeImages[7], items[7].ID, "==to is inclusive")

	// open-ended from: cluster + everything after (the base photos), but never
	// the NULL-corrected image
	from = start
	openFrom := params()
	openFrom.FromCapturedAtCorrected = &from
	_, total, err = repo.GetImages(ctx, openFrom)
	require.NoError(t, err)
	assert.Equal(t, 11, total)

	// open-ended to: cluster only — base photos are newer, NULL-corrected excluded
	to = end
	openTo := params()
	openTo.ToCapturedAtCorrected = &to
	_, total, err = repo.GetImages(ctx, openTo)
	require.NoError(t, err)
	assert.Equal(t, 8, total)

	// a bound equal to a single photo's instant selects exactly that photo
	to = start
	firstOnly := params()
	firstOnly.ToCapturedAtCorrected = &to
	items, total, err = repo.GetImages(ctx, firstOnly)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, m.TimeRangeImages[0], items[0].ID)

	// combined with another narrowing predicate of the shared builder (tag
	// AND/exclude predicates use jsonb containment, which the Postgres tier
	// alone supports — their composition with the range is proven by the same
	// AND-append there). The pattern is deliberately narrower than the range:
	// matching all 8 would also be what an implementation that DROPPED the
	// Search predicate returns, so the AND would stay unproven.
	from = start
	searched := params()
	searched.FromCapturedAtCorrected = &start
	searched.ToCapturedAtCorrected = &end
	pattern := "FSG_9002"
	searched.Search = &pattern
	items, total, err = repo.GetImages(ctx, searched)
	require.NoError(t, err)
	assert.Equal(t, 1, total, "range AND name pattern")
	assert.Equal(t, m.TimeRangeImages[2], items[0].ID)
}

// The Time popover's slider domain: MIN/MAX capturedAtCorrected under the
// shared filter, with any From/To bounds stripped (the range being edited must
// not shift its own domain).
func TestGetImageTimeBounds(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	params := func() *repository.GetImageParameters {
		return &repository.GetImageParameters{ProjectID: m.Project}
	}

	// unbounded project span: earliest = midnight-cluster start, latest = the
	// newest base photo (base instants are ~referenceNow, after the cluster)
	full, err := repo.GetImageTimeBounds(ctx, params())
	require.NoError(t, err)
	require.NotNil(t, full.Min)
	require.NotNil(t, full.Max)
	assert.True(t, full.Min.Equal(m.TimeRangeStart), "min is the cluster start")
	newestBase, err := repo.Client.Image.Get(ctx, m.Images[2])
	require.NoError(t, err)
	assert.True(t, full.Max.Equal(*newestBase.CapturedAtCorrected))

	// narrowed by search: exactly the cluster edges
	pattern := "FSG_90"
	searched := params()
	searched.Search = &pattern
	clusterBounds, err := repo.GetImageTimeBounds(ctx, searched)
	require.NoError(t, err)
	assert.True(t, clusterBounds.Min.Equal(m.TimeRangeStart))
	assert.True(t, clusterBounds.Max.Equal(m.TimeRangeEnd))

	// strip proof: From/To bounds never influence the result
	from, to := m.TimeRangeStart.Add(time.Hour), m.TimeRangeEnd.Add(48*time.Hour)
	bounded := params()
	bounded.FromCapturedAtCorrected = &from
	bounded.ToCapturedAtCorrected = &to
	stripped, err := repo.GetImageTimeBounds(ctx, bounded)
	require.NoError(t, err)
	assert.True(t, stripped.Min.Equal(m.TimeRangeStart), "From ignored")
	assert.True(t, stripped.Max.Equal(*full.Max), "To ignored")

	// a filter matching nothing yields nil sides
	none := params()
	none.Search = &[]string{"no-such-photo"}[0]
	empty, err := repo.GetImageTimeBounds(ctx, none)
	require.NoError(t, err)
	assert.Nil(t, empty.Min)
	assert.Nil(t, empty.Max)
}

// GetImageTimeTicks returns sampled timestamps for the slider density strip.
// The range is always STRIPPED (same as bounds), and only corrected images
// contribute. For ≤ maxTicks images every position is returned; above that
// the list is linearly downsampled.
func TestGetImageTimeTicks(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	params := func() *repository.GetImageParameters {
		return &repository.GetImageParameters{
			ProjectID: m.Project,
		}
	}

	// full set: 8 midnight cluster + 3 base = 11 corrected images
	// maxTicks=200 → all 11 returned
	ticks, err := repo.GetImageTimeTicks(ctx, params(), 200)
	require.NoError(t, err)
	require.NotNil(t, ticks)
	assert.Len(t, ticks, 11, "all corrected images returned when below maxTicks")

	// timestamps must be sorted ascending
	for i := 1; i < len(ticks); i++ {
		assert.False(t, ticks[i].Before(ticks[i-1]), "ticks must be sorted ascending")
	}

	// range is stripped: adding from/to should NOT narrow the ticks
	from, to := m.TimeRangeStart, m.TimeRangeEnd
	bounded := params()
	bounded.FromCapturedAtCorrected = &from
	bounded.ToCapturedAtCorrected = &to
	boundedTicks, err := repo.GetImageTimeTicks(ctx, bounded, 200)
	require.NoError(t, err)
	require.NotNil(t, boundedTicks)
	assert.Len(t, boundedTicks, 11, "time range stripped — same count as unbounded")

	// linear sampling: maxTicks=5 → only 5 timestamps returned
	sampled, err := repo.GetImageTimeTicks(ctx, params(), 5)
	require.NoError(t, err)
	require.Len(t, sampled, 5, "downsampled to maxTicks")
	// The strip must span the whole range: the oldest AND the newest match are
	// always sampled, everything between is spread evenly. Asserting the exact
	// endpoints is what catches a sampler that silently drops the right end.
	assert.True(t, sampled[0].Equal(ticks[0]), "oldest tick preserved")
	assert.True(t, sampled[len(sampled)-1].Equal(ticks[len(ticks)-1]), "newest tick preserved")
	for i := 1; i < len(sampled); i++ {
		assert.False(t, sampled[i].Before(sampled[i-1]), "sampled ticks stay ascending")
		assert.NotEqual(t, sampled[i], sampled[i-1], "sampled ticks are distinct")
	}

	// filter matching nothing returns nil
	none := params()
	none.Search = &[]string{"no-such-photo"}[0]
	empty, err := repo.GetImageTimeTicks(ctx, none, 200)
	require.NoError(t, err)
	assert.Nil(t, empty)
}

// A degenerate domain: 40 photos sharing ONE capture second, plus one clearly
// older and one clearly newer. Every interior sample lands in the tie group, so
// this pins the property that actually matters for the strip — both extremes
// survive a domain whose interior has no ordering of its own.
//
// It is NOT a guard for the sampler's `id` tiebreak: SQLite's unspecified order
// within a tie group is stable across identical query plans, so dropping the
// tiebreak still passes here. The tiebreak is there so each probe is
// deterministic by construction rather than by luck of the plan; two identical
// calls agreeing (asserted below) is a smoke test for that, nothing more.
func TestGetImageTimeTicksSurvivesADegenerateDomain(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	// 40 photos all sharing ONE instant, plus one clearly older and one clearly
	// newer, so a duplicate or an out-of-order seek is unmistakable.
	shared := m.TimeRangeStart.Add(-90 * time.Minute)
	newest := m.TimeRangeEnd.Add(90 * time.Minute)
	ids := make([]string, 0, 42)
	for i := range 40 {
		img, err := repo.Client.Image.Create().
			SetFileName(fmt.Sprintf("TIE_%03d.jpg", i)).
			SetComputedFileName(fmt.Sprintf("FSG_TIE%03d.jpg", i)).
			SetStorageId(fmt.Sprintf("tie%08d", i)).
			SetSize(1024).
			SetWidth(6000).
			SetHeight(4000).
			SetCapturedAt(shared.Add(-seed.Drift)).
			SetCapturedAtCorrected(shared).
			SetUserID(m.Users["projectEditor"]).
			SetUploadID(m.Upload).
			SetProjectID(m.Project).
			SetCameraID(m.Cameras["fresh"]).
			Save(ctx)
		require.NoError(t, err)
		ids = append(ids, img.ID)
	}
	for _, c := range []struct {
		name string
		at   time.Time
	}{{"older", shared.Add(-time.Hour)}, {"newest", newest}} {
		img, err := repo.Client.Image.Create().
			SetFileName("TIE_" + c.name + ".jpg").
			SetComputedFileName("FSG_TIE_" + c.name + ".jpg").
			SetStorageId("tie_" + c.name).
			SetSize(1024).
			SetWidth(6000).
			SetHeight(4000).
			SetCapturedAt(c.at.Add(-seed.Drift)).
			SetCapturedAtCorrected(c.at).
			SetUserID(m.Users["projectEditor"]).
			SetUploadID(m.Upload).
			SetProjectID(m.Project).
			SetCameraID(m.Cameras["fresh"]).
			Save(ctx)
		require.NoError(t, err)
		ids = append(ids, img.ID)
	}

	params := func() *repository.GetImageParameters {
		return &repository.GetImageParameters{ProjectID: m.Project, IDs: ids}
	}

	// Sampled far below the row count so the seek path runs against the ties.
	sampled, err := repo.GetImageTimeTicks(ctx, params(), 8)
	require.NoError(t, err)
	require.Len(t, sampled, 8, "downsampled to maxTicks")
	for i := 1; i < len(sampled); i++ {
		assert.False(t, sampled[i].Before(sampled[i-1]), "sampled ticks stay ascending across a tie group")
	}
	// Both extremes survive, even though every interior sample lands in the tie.
	assert.True(t, sampled[0].Equal(shared.Add(-time.Hour)), "oldest match sampled")
	assert.True(t, sampled[len(sampled)-1].Equal(newest), "newest match sampled")

	// Two identical calls agree (see the func comment for how weak this is).
	again, err := repo.GetImageTimeTicks(ctx, params(), 8)
	require.NoError(t, err)
	assert.Equal(t, sampled, again, "repeated sampling agrees")
}
