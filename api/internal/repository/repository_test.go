package repository_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/auditlog"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/imagetagassignment"
	"github.com/shutterbase/shutterbase/internal/authorization"
	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/seed"
	"github.com/shutterbase/shutterbase/internal/util"
)

// testRepo builds a Repository over a fresh SQLite db via the real boot path
// (Schema.Create). isPostgres() is false here, so the .ForUpdate() branch is
// skipped — the Postgres row-lock path is asserted in the e2e tier.
func testRepo(t *testing.T) *repository.Repository {
	t.Helper()
	conn, err := database.NewConnection(&database.Options{
		DatabaseType: "sqlite",
		File:         filepath.Join(t.TempDir(), "repo.db"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	repo, err := repository.NewRepository(&repository.Options{DatabaseConnection: conn})
	require.NoError(t, err)
	return repo
}

func seededRepo(t *testing.T) (*repository.Repository, *seed.Manifest) {
	t.Helper()
	repo := testRepo(t)
	m, err := seed.Seed(context.Background(), repo.Client, time.Now())
	require.NoError(t, err)
	return repo, m
}

// auditCount waits for the async (safeGo) audit row to land and returns the count
// for one object.
func auditCount(t *testing.T, repo *repository.Repository, objectType, action, objectID string) int {
	t.Helper()
	ctx := context.Background()
	return repo.Client.AuditLog.Query().Where(
		auditlog.ObjectType(objectType),
		auditlog.Action(action),
		auditlog.ObjectId(objectID),
	).CountX(ctx)
}

func TestProjectCRUD(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	p, err := repo.CreateProject(ctx, &repository.CreateProjectParameters{
		Name: "P1", Description: "d", Copyright: "c", CopyrightReference: "r",
		LocationName: "ln", LocationCode: "lc", LocationCity: "city",
	})
	require.NoError(t, err)
	assert.Len(t, p.ID, 15)

	got, err := repo.GetProject(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "P1", got.Name)

	// audit row written on create (async).
	require.Eventually(t, func() bool {
		return auditCount(t, repo, "project", "create", p.ID) == 1
	}, 2*time.Second, 10*time.Millisecond, "create must emit one audit row")

	require.NoError(t, repo.DeleteProject(ctx, p.ID))
	_, err = repo.GetProject(ctx, p.ID)
	assert.Error(t, err)
}

// "provided" semantics: only the non-nil pointer field is changed; others untouched.
func TestUpdateProvidedSemantics(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)
	p, err := repo.CreateProject(ctx, &repository.CreateProjectParameters{
		Name: "Orig", Description: "origdesc", Copyright: "c", CopyrightReference: "r",
		LocationName: "ln", LocationCode: "lc", LocationCity: "city",
	})
	require.NoError(t, err)

	updated, err := repo.UpdateProject(ctx, p.ID, &repository.UpdateProjectParameters{
		Name: util.StringPointer("Renamed"),
	})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", updated.Name)
	assert.Equal(t, "origdesc", updated.Description, "unprovided field must be untouched")

	require.Eventually(t, func() bool {
		return auditCount(t, repo, "project", "update", p.ID) == 1
	}, 2*time.Second, 10*time.Millisecond)
}

// No-op update rolls back and writes NO audit row.
func TestNoOpUpdateRollsBackNoAudit(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)
	p, err := repo.CreateProject(ctx, &repository.CreateProjectParameters{
		Name: "Same", Description: "d", Copyright: "c", CopyrightReference: "r",
		LocationName: "ln", LocationCode: "lc", LocationCity: "city",
	})
	require.NoError(t, err)
	before := p.UpdatedAt

	out, err := repo.UpdateProject(ctx, p.ID, &repository.UpdateProjectParameters{
		Name: util.StringPointer("Same"), // identical -> no change
	})
	require.NoError(t, err)
	assert.Equal(t, before.UnixNano(), out.UpdatedAt.UnixNano(), "no-op must not bump updatedAt")

	// Give any (erroneous) audit goroutine time to run, then assert none.
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, auditCount(t, repo, "project", "update", p.ID), "no-op update must not audit")
}

// Off-allowlist sort is rejected; valid sort works.
func TestSortAllowlist(t *testing.T) {
	ctx := context.Background()
	repo, _ := seededRepo(t)

	_, _, err := repo.GetProjects(ctx, &repository.GetProjectParameters{
		PaginationParameters: &repository.PaginationParameters{Sort: "description"},
	})
	assert.ErrorIs(t, err, repository.ErrInvalidSort, "off-allowlist sort rejected")

	items, total, err := repo.GetProjects(ctx, &repository.GetProjectParameters{
		PaginationParameters: &repository.PaginationParameters{Sort: "name", Order: "asc"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, items, 1)

	// empty sort falls back to the default (createdAt) — still valid.
	_, _, err = repo.GetProjects(ctx, &repository.GetProjectParameters{
		PaginationParameters: &repository.PaginationParameters{},
	})
	require.NoError(t, err)
}

func TestPaginationLimitOffset(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)
	for i := 0; i < 5; i++ {
		_, err := repo.CreateRole(ctx, &repository.CreateRoleParameters{
			Key: "role" + string(rune('a'+i)), Description: "d",
		})
		require.NoError(t, err)
	}
	items, total, err := repo.GetRoles(ctx, &repository.GetRoleParameters{
		PaginationParameters: &repository.PaginationParameters{Limit: 2, Offset: 0, Sort: "key", Order: "asc"},
	})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, items, 2)
}

// IDs restricts the gallery query to the given ids (person filter).
func TestGetImagesIDsFilter(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)
	items, total, err := repo.GetImages(ctx, &repository.GetImageParameters{
		ProjectID:            m.Project,
		IDs:                  []string{m.Images[0], m.Images[1]},
		PaginationParameters: &repository.PaginationParameters{Limit: 10},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, items, 2)
}

// SetImageTags rebuilds the denormalized list; idempotent assignment.
func TestAssignmentIdempotentAndDenormalization(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	img := m.Images[0]
	podium := m.Tags["Podium"]
	defaultTag := m.Tags["Default"]

	// First assignment of Podium -> created.
	a1, created, err := repo.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{
		ImageID: img, ImageTagID: podium, Type: imagetagassignment.TypeManual,
	})
	require.NoError(t, err)
	assert.True(t, created)

	// Denormalized list now contains both Default and Podium.
	got, err := repo.GetImage(ctx, img)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{defaultTag, podium}, got.ImageTags)

	// Re-assign same pair -> idempotent (200, not 409), same row, not created.
	a2, created, err := repo.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{
		ImageID: img, ImageTagID: podium, Type: imagetagassignment.TypeManual,
	})
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, a1.ID, a2.ID)

	// Still exactly two assignments on the image.
	cnt := repo.Client.ImageTagAssignment.Query().Where(imagetagassignment.ImageID(img)).CountX(ctx)
	assert.Equal(t, 2, cnt)
}

// Repair-on-tag-delete: deleting a tag strips it from images.imageTags and drops
// its assignments, leaving the other tags intact.
func TestRepairOnTagDelete(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	img := m.Images[0]
	podium := m.Tags["Podium"]
	defaultTag := m.Tags["Default"]

	_, _, err := repo.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{
		ImageID: img, ImageTagID: podium, Type: imagetagassignment.TypeManual,
	})
	require.NoError(t, err)

	// Delete the Default tag -> repaired list keeps Podium only.
	require.NoError(t, repo.DeleteImageTag(ctx, defaultTag))

	got, err := repo.GetImage(ctx, img)
	require.NoError(t, err)
	assert.Equal(t, []string{podium}, got.ImageTags, "deleted tag stripped from denormalized list")

	// The Default tag's assignments are gone for every seeded image.
	left := repo.Client.ImageTagAssignment.Query().Where(imagetagassignment.ImageTagID(defaultTag)).CountX(ctx)
	assert.Equal(t, 0, left)
}

// Two-character camera names ("R5", "Z6") are legitimate — the old MinLen(3)
// rejected them with an opaque 400 (field report, 2026-07-25). Empty stays out.
func TestCameraShortNameAllowed(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	cam, err := repo.CreateCamera(ctx, &repository.CreateCameraParameters{Name: "R5", UserID: m.Users["projectEditor"]})
	require.NoError(t, err)
	assert.Equal(t, "R5", cam.Name)

	_, err = repo.CreateCamera(ctx, &repository.CreateCameraParameters{Name: "", UserID: m.Users["projectEditor"]})
	assert.Error(t, err, "an empty name is still rejected")
}

func TestUserCRUDUUID(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	u, err := repo.CreateUser(ctx, &repository.CreateUserParameters{
		Username: "neo", FirstName: "Thomas", LastName: "Anderson",
		Email: util.StringPointer("neo@example.test"), Active: util.BoolPointer(true),
	})
	require.NoError(t, err)

	got, err := repo.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "neo", got.Username)

	upd, err := repo.UpdateUser(ctx, u.ID, &repository.UpdateUserParameters{
		FirstName: util.StringPointer("Neo"),
	})
	require.NoError(t, err)
	assert.Equal(t, "Neo", upd.FirstName)
	assert.Equal(t, "Anderson", upd.LastName)

	require.Eventually(t, func() bool {
		return auditCount(t, repo, "user", "create", u.ID.String()) == 1 &&
			auditCount(t, repo, "user", "update", u.ID.String()) == 1
	}, 2*time.Second, 10*time.Millisecond)

	require.NoError(t, repo.DeleteUser(ctx, u.ID))
}

// PocketBase-hook parity: a new user without an explicit copyright tag gets
// their last name as the default; an explicit tag always wins.
func TestCreateUserDefaultsCopyrightTagToLastName(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	defaulted, err := repo.CreateUser(ctx, &repository.CreateUserParameters{
		Username: "trinity", FirstName: "Trinity", LastName: "Moss",
	})
	require.NoError(t, err)
	assert.Equal(t, "Moss", defaulted.CopyrightTag)

	emptied, err := repo.CreateUser(ctx, &repository.CreateUserParameters{
		Username: "smith", FirstName: "Agent", LastName: "Smith",
		CopyrightTag: util.StringPointer(""),
	})
	require.NoError(t, err)
	assert.Equal(t, "Smith", emptied.CopyrightTag)

	explicit, err := repo.CreateUser(ctx, &repository.CreateUserParameters{
		Username: "oracle", FirstName: "The", LastName: "Oracle",
		CopyrightTag: util.StringPointer("ORCL"),
	})
	require.NoError(t, err)
	assert.Equal(t, "ORCL", explicit.CopyrightTag)
}

// GetUploadMetrics runs a hand-built join through the ent SQL builder — the one
// query in the repository that is not fully typed. This pins its behavior
// against a real database (an aliasing bug here produced a runtime SQL error
// that no build or type check could catch).
func TestGetUploadMetrics(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	up, err := repo.GetUpload(ctx, m.Upload)
	require.NoError(t, err)

	metrics, err := repo.GetUploadMetrics(ctx, []*ent.Upload{up})
	require.NoError(t, err)
	require.Contains(t, metrics, up.ID)
	assert.Equal(t, len(m.Images), metrics[up.ID].ImageCount, "every seeded image counts")
	assert.Equal(t, 0, metrics[up.ID].TagCount, "seed applies no manual tags")

	// One manual assignment on one image -> counted, and the derived rate follows.
	_, _, err = repo.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{
		ImageID: m.Images[0], ImageTagID: m.Tags["Podium"], Type: imagetagassignment.TypeManual,
	})
	require.NoError(t, err)

	metrics, err = repo.GetUploadMetrics(ctx, []*ent.Upload{up})
	require.NoError(t, err)
	assert.Equal(t, 1, metrics[up.ID].TagCount)
	assert.InDelta(t, 1.0/float64(len(m.Images)), metrics[up.ID].TagsPerImage, 0.0001)

	// errorCount is live state: it follows the review error tag on and off.
	errorTag := repo.Client.ImageTag.Create().
		SetName(authorization.ReviewErrorTagName).SetDescription("tagging error").SetType(imagetag.TypeCustom).SetProjectID(m.Project).SaveX(ctx)
	assignment, _, err := repo.CreateImageTagAssignment(ctx, &repository.CreateImageTagAssignmentParameters{
		ImageID: m.Images[0], ImageTagID: errorTag.ID, Type: imagetagassignment.TypeManual,
	})
	require.NoError(t, err)

	metrics, err = repo.GetUploadMetrics(ctx, []*ent.Upload{up})
	require.NoError(t, err)
	assert.Equal(t, 1, metrics[up.ID].ErrorCount)

	require.NoError(t, repo.DeleteImageTagAssignment(ctx, assignment.ID))
	metrics, err = repo.GetUploadMetrics(ctx, []*ent.Upload{up})
	require.NoError(t, err)
	assert.Equal(t, 0, metrics[up.ID].ErrorCount, "clearing the tag lowers the count again")

	// An empty set must not build a query at all.
	empty, err := repo.GetUploadMetrics(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

// The active-tagging accumulator read-modify-writes the upload row; assert the
// idle-gap heuristic lands.
func TestUploadTaggingMetricAccumulation(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)
	t0 := time.Now()

	require.NoError(t, repo.RecordTaggingActivity(ctx, m.Upload, t0))
	require.NoError(t, repo.RecordTaggingActivity(ctx, m.Upload, t0.Add(20*time.Second)))
	require.NoError(t, repo.RecordTaggingActivity(ctx, m.Upload, t0.Add(3*time.Hour)))

	up, err := repo.GetUpload(ctx, m.Upload)
	require.NoError(t, err)
	assert.Equal(t, 20, up.TaggingSeconds, "the three-hour break is not working time")
}
