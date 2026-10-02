package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// A photo that sits ALONE in time must still appear on the density strip.
//
// The strip used to sample by row offset, which is a histogram of the photo
// distribution: with 15 000 photos inside one 15-second burst and a handful of
// photos hours away, all 200 marks landed inside the burst and every lone photo
// vanished. Measured: 5 of 5 dropped. Time-bucketing alone did not fix it
// either — a lone photo 1.5h past the burst shares a 15h bucket with it, and a
// bucket emits one median — so there is an explicit isolation pass as well.
func TestLonePhotosSurviveTheDensityStrip(t *testing.T) {
	repo, m := seededRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	lone := []time.Duration{90 * time.Minute, 5 * time.Hour, 20 * time.Hour, 30 * time.Hour, 47 * time.Hour}

	builders := make([]*ent.ImageCreate, 0, 15005)
	idx := 0
	for i := 0; i < 15000; i++ {
		idx++
		builders = append(builders, repo.Client.Image.Create().
			SetFileName(fmt.Sprintf("burst%05d.jpg", i)).
			SetStorageId(fmt.Sprintf("s%06d", idx)).SetSize(1).SetProjectID(m.Project).SetUserID(m.Users["projectEditor"]).SetUploadID(m.Upload).SetCameraID(m.Cameras["fresh"]).
			SetCapturedAtCorrected(base.Add(time.Duration(i)*time.Millisecond)))
	}
	for i, d := range lone {
		idx++
		builders = append(builders, repo.Client.Image.Create().
			SetFileName(fmt.Sprintf("lone%d.jpg", i)).
			SetStorageId(fmt.Sprintf("s%06d", idx)).SetSize(1).SetProjectID(m.Project).SetUserID(m.Users["projectEditor"]).SetUploadID(m.Upload).SetCameraID(m.Cameras["fresh"]).
			SetCapturedAtCorrected(base.Add(d)))
	}
	// SQLite caps bound variables per statement, so chunk like the seeder does.
	for start := 0; start < len(builders); start += 500 {
		end := min(start+500, len(builders))
		if err := repo.Client.Image.CreateBulk(builders[start:end]...).Exec(ctx); err != nil {
			t.Fatalf("chunk %d: %v", start, err)
		}
	}

	ticks, err := repo.GetImageTimeTicks(ctx, &repository.GetImageParameters{ProjectID: m.Project}, stripBudget)
	if err != nil {
		t.Fatal(err)
	}
	require.LessOrEqual(t, len(ticks), stripBudget, "never more than maxTicks")
	for i, d := range lone {
		at := base.Add(d)
		found := false
		for _, tk := range ticks {
			if tk.UTC().Truncate(time.Second).Equal(at) {
				found = true
				break
			}
		}
		assert.True(t, found, "lone photo %d at +%s is not on the density strip", i, d)
	}
	// The burst must still be represented. A floor of 1 is useless here: the burst
	// is 15 000 rows inside one time bucket, so a single mark satisfies it and the
	// strip could collapse to a bare outlier list without failing. The burst spans
	// one bucketWidth, so the honest expectation is the one mark plus the pinned
	// endpoints that land inside it.
	burstMarks := 0
	for _, tk := range ticks {
		if tk.Sub(base) < time.Minute {
			burstMarks++
		}
	}
	assert.GreaterOrEqual(t, burstMarks, 2, "the 15k-photo burst contributes marks, not just one")
	// Strictly ascending, not merely non-decreasing: burst photos share a second
	// and two marks at the same pixel are one mark drawn twice.
	assertStrictlyAscending(t, ticks, "lone-photo strip")
}

// TestDensityResolutionSurvivesWhenNothingIsIsolated guards the budget split.
//
// The sampler used to reserve half the tick budget for lone photos before
// knowing whether any existed, so a smooth gallery with nothing isolated threw
// away half its density resolution for nothing: 15 000 photos spread evenly over
// a week produced ~101 marks instead of ~200. Every project without outliers
// paid that. Spacing here is ~40s end to end, so no row is isolated.
func TestDensityResolutionSurvivesWhenNothingIsIsolated(t *testing.T) {
	repo, m := seededRepo(t)
	ctx := context.Background()
	// Anchored to now, not a fixed date: seededRepo already put fixture photos at
	// now, so a fixed 2026-08-01 start would stretch the domain to ~62 days and
	// this block would fill 3 of the 198 buckets instead of all of them. Measured:
	// 27 marks that way. Ending at now keeps the span at 7 days.
	base := time.Now().Add(-7 * 24 * time.Hour).Truncate(time.Minute)
	const total = 15000
	step := 7 * 24 * time.Hour / total

	builders := make([]*ent.ImageCreate, 0, total)
	for i := 0; i < total; i++ {
		builders = append(builders, repo.Client.Image.Create().
			SetFileName(fmt.Sprintf("even%05d.jpg", i)).
			SetStorageId(fmt.Sprintf("ev%06d", i)).SetSize(1).SetProjectID(m.Project).
			SetUserID(m.Users["projectEditor"]).SetUploadID(m.Upload).SetCameraID(m.Cameras["fresh"]).
			SetCapturedAtCorrected(base.Add(time.Duration(i)*step)))
	}
	insertChunked(t, ctx, repo, builders)

	ticks, err := repo.GetImageTimeTicks(ctx, &repository.GetImageParameters{ProjectID: m.Project}, stripBudget)
	require.NoError(t, err)
	// Every budget slot except the two pinned endpoints should carry a mark: the
	// gallery is uniform, so every time bucket is non-empty.
	assert.GreaterOrEqual(t, len(ticks), stripBudget-10,
		"a gallery with no isolated photos keeps the full density resolution")
	assertStrictlyAscending(t, ticks, "even strip")
}

// TestIsolatedPhotosStaySpreadAcrossTheDomainWhenOverBudget guards the cap.
//
// When isolated photos outnumber the budget, thinning them by TIME order is the
// whole point — dropping the tail wholesale hides a region of the timeline, which
// is the failure the isolation pass exists to prevent. The earlier cap broke at
// the first N found, so with 400 isolated photos only the earliest 99 were shown
// and everything later was invisible.
func TestIsolatedPhotosStaySpreadAcrossTheDomainWhenOverBudget(t *testing.T) {
	repo, m := seededRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	// 400 photos alone, three hours apart across 50 days...
	builders := make([]*ent.ImageCreate, 0, 15400)
	idx := 0
	for i := 0; i < 400; i++ {
		idx++
		builders = append(builders, repo.Client.Image.Create().
			SetFileName(fmt.Sprintf("iso%04d.jpg", i)).
			SetStorageId(fmt.Sprintf("is%06d", idx)).SetSize(1).SetProjectID(m.Project).
			SetUserID(m.Users["projectEditor"]).SetUploadID(m.Upload).SetCameraID(m.Cameras["fresh"]).
			SetCapturedAtCorrected(base.Add(time.Duration(i)*3*time.Hour)))
	}
	// ...then a dense final day, so density and isolation both want budget.
	denseStart := base.Add(60 * 24 * time.Hour)
	for i := 0; i < 15000; i++ {
		idx++
		builders = append(builders, repo.Client.Image.Create().
			SetFileName(fmt.Sprintf("dense%05d.jpg", i)).
			SetStorageId(fmt.Sprintf("de%06d", idx)).SetSize(1).SetProjectID(m.Project).
			SetUserID(m.Users["projectEditor"]).SetUploadID(m.Upload).SetCameraID(m.Cameras["fresh"]).
			SetCapturedAtCorrected(denseStart.Add(time.Duration(i)*time.Second)))
	}
	insertChunked(t, ctx, repo, builders)

	ticks, err := repo.GetImageTimeTicks(ctx, &repository.GetImageParameters{ProjectID: m.Project}, stripBudget)
	require.NoError(t, err)
	require.LessOrEqual(t, len(ticks), stripBudget, "never more than maxTicks")
	assertStrictlyAscending(t, ticks, "over-budget strip")

	marked := make(map[time.Time]struct{}, len(ticks))
	for _, tk := range ticks {
		marked[tk.UTC().Truncate(time.Second)] = struct{}{}
	}

	// Counting marks per quarter of the isolated photos' range, rather than
	// asserting the very last one is marked. Asserting on a single photo is
	// flaky: a lone photo can be picked up incidentally as its bucket's median,
	// and that happened to happen for the 400th photo when this test was first
	// written — it passed against the very code it was meant to catch. The
	// distribution is the invariant: thinning by time leaves each quarter with
	// roughly its share, while breaking at the first N leaves the last quarter
	// with almost nothing.
	quarter := make([]int, 4)
	for i := 0; i < 400; i++ {
		at := base.Add(time.Duration(i) * 3 * time.Hour).Truncate(time.Second)
		if _, ok := marked[at]; ok {
			quarter[i/100]++
		}
	}
	t.Logf("marked isolated photos per quarter: %v", quarter)
	assert.Greater(t, quarter[0], 0, "the earliest quarter has marked isolated photos")
	assert.GreaterOrEqual(t, quarter[3], quarter[0]/2,
		"the last quarter is thinned with the rest, not dropped: %v", quarter)

	// And the dense day still gets its share, which is what the density floor buys.
	denseMarks := 0
	for tk := range marked {
		if !tk.Before(denseStart) {
			denseMarks++
		}
	}
	assert.GreaterOrEqual(t, denseMarks, 1, "the dense day keeps marks even with 400 isolated photos")
}

// stripBudget mirrors maxTimeTicks in images_controller.go, which is where the
// real value comes from. Duplicated rather than imported so the test still
// pins a literal: if the controller changes, this test should fail loudly.
const stripBudget = 200

func insertChunked(t *testing.T, ctx context.Context, repo *repository.Repository, builders []*ent.ImageCreate) {
	t.Helper()
	for start := 0; start < len(builders); start += 500 {
		end := min(start+500, len(builders))
		require.NoError(t, repo.Client.Image.CreateBulk(builders[start:end]...).Exec(ctx),
			"chunk %d", start)
	}
}

func assertStrictlyAscending(t *testing.T, ticks []time.Time, what string) {
	t.Helper()
	for i := 1; i < len(ticks); i++ {
		assert.True(t, ticks[i].After(ticks[i-1]),
			"%s: tick %d (%s) must be strictly after tick %d (%s)",
			what, i, ticks[i], i-1, ticks[i-1])
	}
}
