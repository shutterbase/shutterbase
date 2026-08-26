package repository_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/schema"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/seed"
)

// timelineImages returns the seeded BASE images (the midnight fixture cluster
// is excluded — its instants sit strictly before the base window, so tracks
// built on this axis cover exactly what the assertions expect) sorted by
// corrected capture time — the axis ApplyUploadTimeline reconciles on.
func timelineImages(t *testing.T, repo *repository.Repository, m *seed.Manifest) []timelineImg {
	t.Helper()
	base := m.Images[:len(m.Images)-len(m.TimeRangeImages)]
	rows, err := repo.Client.Image.Query().Where(image.UploadID(m.Upload), image.IDIn(base...)).All(context.Background())
	require.NoError(t, err)
	out := make([]timelineImg, 0, len(rows))
	for _, r := range rows {
		require.NotNil(t, r.CapturedAtCorrected)
		out = append(out, timelineImg{id: r.ID, at: *r.CapturedAtCorrected})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].at.Before(out[j].at) })
	return out
}

type timelineImg struct {
	id string
	at time.Time
}

// scheduledTags returns the tag ids of type=scheduled assignments on one image.
func scheduledTags(t *testing.T, repo *repository.Repository, imageID string) []string {
	t.Helper()
	ids, err := repo.Client.ImageTagAssignment.Query().
		Where(imagetagassignment.ImageID(imageID), imagetagassignment.TypeEQ(imagetagassignment.TypeScheduled)).
		Select(imagetagassignment.FieldImageTagID).Strings(context.Background())
	require.NoError(t, err)
	return ids
}

func TestApplyUploadTimelineFreeTagTrack(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)
	imgs := timelineImages(t, repo, m)
	require.Len(t, imgs, 3)
	podium := m.Tags["Podium"]

	// Window [img0, img2) covers the first two images.
	res, err := repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{TagID: podium, Start: imgs[0].at, End: imgs[2].at, Enabled: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, res.Created)
	assert.Equal(t, 0, res.Deleted)
	assert.Equal(t, []string{podium}, scheduledTags(t, repo, imgs[0].id))
	assert.Equal(t, []string{podium}, scheduledTags(t, repo, imgs[1].id))
	assert.Empty(t, scheduledTags(t, repo, imgs[2].id))
	require.Len(t, res.Upload.Timeline, 1, "editor state persisted")

	// Denormalized read model rebuilt.
	img0, err := repo.GetImage(ctx, imgs[0].id)
	require.NoError(t, err)
	assert.Contains(t, img0.ImageTags, podium)

	// Narrowing the window revokes only what fell out.
	res, err = repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{TagID: podium, Start: imgs[0].at, End: imgs[1].at, Enabled: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, res.Created)
	assert.Equal(t, 1, res.Deleted)
	assert.Empty(t, scheduledTags(t, repo, imgs[1].id))
	assert.Equal(t, []string{podium}, scheduledTags(t, repo, imgs[0].id))

	// A disabled track applies nothing (and revokes what it covered before).
	res, err = repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{TagID: podium, Start: imgs[0].at, End: imgs[1].at, Enabled: false},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Deleted)
	assert.Empty(t, scheduledTags(t, repo, imgs[0].id))
}

func TestApplyUploadTimelineProtectsManualTags(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)
	imgs := timelineImages(t, repo, m)
	podium := m.Tags["Podium"]

	// Photographer hand-tagged image 0 with Podium.
	_, _, err := repo.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{
		ImageID: imgs[0].id, ImageTagID: podium, Type: imagetagassignment.TypeManual,
	})
	require.NoError(t, err)

	// A covering track must not duplicate the manual row…
	res, err := repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{TagID: podium, Start: imgs[0].at, End: imgs[0].at.Add(time.Millisecond), Enabled: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, res.Created, "existing manual assignment is left alone")

	// …and clearing the timeline must not delete it either.
	res, err = repo.ApplyUploadTimeline(ctx, m.Upload, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, res.Deleted, "manual rows are never the timeline's to revoke")
	n := repo.Client.ImageTagAssignment.Query().Where(
		imagetagassignment.ImageID(imgs[0].id),
		imagetagassignment.ImageTagID(podium),
	).CountX(ctx)
	assert.Equal(t, 1, n)
}

func TestApplyUploadTimelineScheduleItemTrack(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)
	imgs := timelineImages(t, repo, m)
	podium := m.Tags["Podium"]

	item, err := repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Endurance", Start: imgs[0].at, End: imgs[2].at.Add(time.Second),
		ProjectID: m.Project, TagIDs: []string{podium, m.Tags["Default"]},
	})
	require.NoError(t, err)

	// The item's suggestion set fans out onto every covered image. The Default
	// tag already exists as a type=default assignment -> skipped, not duplicated.
	res, err := repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{ScheduleItemID: item.ID, Start: item.Start, End: item.End, Enabled: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 3, res.Created, "podium on all three; default rows pre-exist")

	// Disabling the track revokes exactly those three scheduled rows; the
	// seed's default-type rows survive.
	res, err = repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{ScheduleItemID: item.ID, Start: item.Start, End: item.End, Enabled: false},
	})
	require.NoError(t, err)
	assert.Equal(t, 3, res.Deleted)
	for _, img := range imgs {
		assert.Empty(t, scheduledTags(t, repo, img.id))
	}
	got, err := repo.GetImage(ctx, imgs[0].id)
	require.NoError(t, err)
	assert.Contains(t, got.ImageTags, m.Tags["Default"], "default-type assignment untouched")
}

func TestApplyUploadTimelineValidation(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)
	imgs := timelineImages(t, repo, m)
	podium := m.Tags["Podium"]
	t0, t1 := imgs[0].at, imgs[0].at.Add(time.Hour)

	item, err := repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "A", Start: t0, End: t1, ProjectID: m.Project,
	})
	require.NoError(t, err)
	item2, err := repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "B", Start: t1, End: t1.Add(time.Hour), ProjectID: m.Project,
	})
	require.NoError(t, err)

	// Both ids set / neither set / end<=start -> structurally invalid.
	_, err = repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{ScheduleItemID: item.ID, TagID: podium, Start: t0, End: t1, Enabled: true},
	})
	assert.ErrorIs(t, err, repository.ErrInvalidTimeline)
	_, err = repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{Start: t0, End: t1, Enabled: true},
	})
	assert.ErrorIs(t, err, repository.ErrInvalidTimeline)
	_, err = repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{TagID: podium, Start: t1, End: t0, Enabled: true},
	})
	assert.ErrorIs(t, err, repository.ErrInvalidTimeline)

	// Unknown schedule item -> invalid.
	_, err = repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{ScheduleItemID: "nope", Start: t0, End: t1, Enabled: true},
	})
	assert.ErrorIs(t, err, repository.ErrInvalidTimeline)

	// Overlapping ENABLED schedule tracks -> mutually exclusive violation;
	// boundary-touching and disabled overlaps are fine.
	_, err = repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{ScheduleItemID: item.ID, Start: t0, End: t1, Enabled: true},
		{ScheduleItemID: item2.ID, Start: t1.Add(-time.Minute), End: t1.Add(time.Hour), Enabled: true},
	})
	assert.ErrorIs(t, err, repository.ErrScheduleOverlap)
	_, err = repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{ScheduleItemID: item.ID, Start: t0, End: t1, Enabled: true},
		{ScheduleItemID: item2.ID, Start: t1, End: t1.Add(time.Hour), Enabled: true},
	})
	assert.NoError(t, err, "touching boundaries do not overlap")
	_, err = repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{ScheduleItemID: item.ID, Start: t0, End: t1, Enabled: true},
		{ScheduleItemID: item2.ID, Start: t0, End: t1, Enabled: false},
	})
	assert.NoError(t, err, "disabled tracks may overlap freely")

	// Foreign tag -> project mismatch.
	other, err := repo.CreateProject(ctx, &repository.CreateProjectParameters{
		Name: "Other", Description: "d", Copyright: "c", CopyrightReference: "r",
		LocationName: "ln", LocationCode: "lc", LocationCity: "city",
	})
	require.NoError(t, err)
	foreign, err := repo.CreateImageTag(ctx, &repository.CreateImageTagParameters{
		Name: "foreign", Description: "d", Type: "manual", ProjectID: other.ID,
	})
	require.NoError(t, err)
	_, err = repo.ApplyUploadTimeline(ctx, m.Upload, []schema.TimelineTrack{
		{TagID: foreign.ID, Start: t0, End: t1, Enabled: true},
	})
	assert.ErrorIs(t, err, repository.ErrTagProjectMismatch)
}
