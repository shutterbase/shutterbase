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

	ticks, err := repo.GetImageTimeTicks(ctx, &repository.GetImageParameters{ProjectID: m.Project}, 200)
	if err != nil {
		t.Fatal(err)
	}
	require.LessOrEqual(t, len(ticks), 200, "never more than maxTicks")
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
	// The burst must still dominate: that is what makes this a density strip and
	// not just a list of outliers.
	burstMarks := 0
	for _, tk := range ticks {
		if tk.Sub(base) < time.Minute {
			burstMarks++
		}
	}
	assert.GreaterOrEqual(t, burstMarks, 1, "the 15k-photo burst is still represented")
	for i := 1; i < len(ticks); i++ {
		assert.False(t, ticks[i].Before(ticks[i-1]), "ticks stay ascending")
	}
}
