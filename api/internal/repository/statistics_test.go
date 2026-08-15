package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent/image"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/internal/repository"
)

// Statistics aggregation on a dedicated project (the seeded one keeps its
// time-relative fixtures out of the assertions). The two bucketed images
// straddle a Berlin midnight in UTC terms, pinning the event-timezone rule:
// 21:30Z and 22:30Z are the same UTC day but different Berlin days.
func TestGetProjectStatistics(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	editor := m.Users["projectEditor"]
	admin := m.Users["admin"]
	camera := m.Cameras["fresh"]

	project, err := repo.Client.Project.Create().
		SetName("Stats Fixture").SetDescription("d").SetCopyright("c").SetCopyrightReference("cr").
		SetLocationName("l").SetLocationCode("lc").SetLocationCity("city").
		Save(ctx)
	require.NoError(t, err)

	upOpen, err := repo.Client.Upload.Create().
		SetName("u-open").SetProjectID(project.ID).SetUserID(editor).SetCameraID(camera).
		Save(ctx)
	require.NoError(t, err)
	_, err = repo.Client.Upload.Create().
		SetName("u-reviewed").SetProjectID(project.ID).SetUserID(editor).SetCameraID(camera).
		SetState(upload.StateReviewed).
		Save(ctx)
	require.NoError(t, err)

	// 2026-08-10 21:30Z = 23:30 Berlin (day 1); 22:30Z = 00:30 Berlin (day 2).
	day1 := time.Date(2026, 8, 10, 21, 30, 0, 0, time.UTC)
	day2 := time.Date(2026, 8, 10, 22, 30, 0, 0, time.UTC)

	img1, err := repo.Client.Image.Create().
		SetFileName("S_0001.jpg").SetStorageId("statsimg00000001").SetSize(100).
		SetCapturedAtCorrected(day1).SetAiStatus(image.AiStatusDone).
		SetUserID(editor).SetUploadID(upOpen.ID).SetProjectID(project.ID).SetCameraID(camera).
		Save(ctx)
	require.NoError(t, err)
	_, err = repo.Client.Image.Create().
		SetFileName("S_0002.jpg").SetStorageId("statsimg00000002").SetSize(200).
		SetCapturedAtCorrected(day2).SetAiStatus(image.AiStatusError).
		SetUserID(editor).SetUploadID(upOpen.ID).SetProjectID(project.ID).SetCameraID(camera).
		Save(ctx)
	require.NoError(t, err)
	// no corrected time -> unbucketed, second photographer
	_, err = repo.Client.Image.Create().
		SetFileName("S_0003.jpg").SetStorageId("statsimg00000003").SetSize(300).
		SetUserID(admin).SetUploadID(upOpen.ID).SetProjectID(project.ID).SetCameraID(camera).
		Save(ctx)
	require.NoError(t, err)

	tag, err := repo.Client.ImageTag.Create().
		SetName("stats-tag").SetDescription("d").SetType(imagetag.TypeManual).SetProjectID(project.ID).
		Save(ctx)
	require.NoError(t, err)
	tag2, err := repo.Client.ImageTag.Create().
		SetName("stats-tag-2").SetDescription("d").SetType(imagetag.TypeManual).SetProjectID(project.ID).
		Save(ctx)
	require.NoError(t, err)

	// manual on Berlin day 2, inferred on Berlin day 1, default excluded entirely.
	_, err = repo.Client.ImageTagAssignment.Create().
		SetType(imagetagassignment.TypeManual).SetImageID(img1.ID).SetImageTagID(tag.ID).
		SetCreatedAt(day2).
		Save(ctx)
	require.NoError(t, err)
	_, err = repo.Client.ImageTagAssignment.Create().
		SetType(imagetagassignment.TypeInferred).SetImageID(img1.ID).SetImageTagID(tag2.ID).
		SetCreatedAt(time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)).
		Save(ctx)
	require.NoError(t, err)

	stats, err := repo.GetProjectStatistics(ctx, project.ID, berlin)
	require.NoError(t, err)

	assert.Equal(t, 3, stats.Totals.Images)
	assert.Equal(t, 2, stats.Totals.Photographers)
	assert.Equal(t, 1, stats.Totals.ManualTags)
	assert.Equal(t, 1, stats.Totals.AiTags)
	assert.Equal(t, int64(600), stats.Totals.StorageBytes)
	assert.Equal(t, 1, stats.Totals.UnbucketedImages)

	require.Len(t, stats.Days, 2, "images straddle a Berlin midnight")
	assert.Equal(t, "2026-08-10", stats.Days[0].Date)
	assert.Equal(t, 1, stats.Days[0].Total)
	assert.Equal(t, 1, stats.Days[0].ByHour[23], "23:30 Berlin")
	assert.Equal(t, "2026-08-11", stats.Days[1].Date)
	assert.Equal(t, 1, stats.Days[1].ByHour[0], "00:30 Berlin")
	assert.Equal(t, 1, stats.Days[0].ByUser[editor.String()])

	require.Len(t, stats.AssignmentsPerDay, 2)
	assert.Equal(t, repository.StatAssignmentDay{Date: "2026-08-10", Manual: 0, AI: 1}, stats.AssignmentsPerDay[0])
	assert.Equal(t, repository.StatAssignmentDay{Date: "2026-08-11", Manual: 1, AI: 0}, stats.AssignmentsPerDay[1])

	assert.Equal(t, repository.StatAiStatus{Done: 1, Error: 1, InFlight: 0, NotQueued: 1}, stats.AiStatus)
	assert.Equal(t, repository.StatUploadStates{Open: 1, Ready: 0, Reviewed: 1}, stats.UploadStates)

	// photographers sorted by count desc: editor (2) before admin (1)
	require.Len(t, stats.Photographers, 2)
	assert.Equal(t, editor.String(), stats.Photographers[0].ID)
	assert.Equal(t, 2, stats.Photographers[0].ImageCount)
	assert.Equal(t, 1, stats.Photographers[1].ImageCount)

	assert.Len(t, stats.Tags, 2, "tag stats scoped to the fixture project")

	// The seeded project keeps its own numbers — no cross-project bleed.
	seedStats, err := repo.GetProjectStatistics(ctx, m.Project, berlin)
	require.NoError(t, err)
	assert.Equal(t, len(m.Images), seedStats.Totals.Images)
}
