package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/util"
)

func TestScheduleItemCRUD(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)
	start := m.ReferenceNow
	end := start.Add(2 * time.Hour)

	item, err := repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Endurance", Description: "the long one", Start: start, End: end,
		Cardinality: 3, ProjectID: m.Project, TagIDs: []string{m.Tags["Podium"]},
	})
	require.NoError(t, err)
	assert.Len(t, item.ID, 15)
	assert.Equal(t, 3, item.Cardinality)
	require.Len(t, item.Edges.Tags, 1, "suggestion tags eager-loaded")
	assert.Equal(t, m.Tags["Podium"], item.Edges.Tags[0].ID)
	assert.Empty(t, item.Edges.Assignees)

	// default cardinality is 1 when none is provided.
	def, err := repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Autocross", Start: start.Add(3 * time.Hour), End: start.Add(4 * time.Hour), ProjectID: m.Project,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, def.Cardinality)

	// provided semantics: only the pointer fields change; tags replace wholesale.
	upd, err := repo.UpdateScheduleItem(ctx, item.ID, &repository.UpdateScheduleItemParameters{
		Title:  util.StringPointer("Endurance+"),
		TagIDs: &[]string{m.Tags["Default"]},
	})
	require.NoError(t, err)
	assert.Equal(t, "Endurance+", upd.Title)
	assert.Equal(t, "the long one", upd.Description, "unprovided field untouched")
	require.Len(t, upd.Edges.Tags, 1)
	assert.Equal(t, m.Tags["Default"], upd.Edges.Tags[0].ID, "tag set replaced")

	require.Eventually(t, func() bool {
		return auditCount(t, repo, "schedule_item", "create", item.ID) == 1 &&
			auditCount(t, repo, "schedule_item", "update", item.ID) == 1
	}, 2*time.Second, 10*time.Millisecond)

	require.NoError(t, repo.DeleteScheduleItem(ctx, item.ID))
	_, err = repo.GetScheduleItem(ctx, item.ID)
	assert.Error(t, err)
}

func TestScheduleItemListOverlapAndMine(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)
	t0 := m.ReferenceNow

	mk := func(title string, startOffset, endOffset time.Duration) string {
		it, err := repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
			Title: title, Start: t0.Add(startOffset), End: t0.Add(endOffset), ProjectID: m.Project,
		})
		require.NoError(t, err)
		return it.ID
	}
	morning := mk("morning", 0, 2*time.Hour)
	noon := mk("noon", 3*time.Hour, 5*time.Hour)
	evening := mk("evening", 6*time.Hour, 8*time.Hour)

	// overlap window [2h30, 6h30) catches noon and evening, not morning.
	from, to := t0.Add(150*time.Minute), t0.Add(390*time.Minute)
	items, total, err := repo.GetScheduleItems(ctx, &repository.GetScheduleItemsParameters{
		ProjectID: m.Project, From: &from, To: &to,
		PaginationParameters: &repository.PaginationParameters{Sort: "start", Order: "asc"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
	assert.Equal(t, noon, items[0].ID)
	assert.Equal(t, evening, items[1].ID)

	// assignee filter: editor covers "morning" only.
	editor := m.Users["projectEditor"]
	_, err = repo.AssignScheduleItemUser(ctx, morning, editor)
	require.NoError(t, err)
	items, total, err = repo.GetScheduleItems(ctx, &repository.GetScheduleItemsParameters{
		ProjectID: m.Project, AssigneeID: &editor,
		PaginationParameters: &repository.PaginationParameters{},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, morning, items[0].ID)
}

func TestScheduleItemAssignIdempotentAndOverbooking(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	item, err := repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Skidpad", Start: m.ReferenceNow, End: m.ReferenceNow.Add(time.Hour),
		Cardinality: 1, ProjectID: m.Project,
	})
	require.NoError(t, err)

	editor := m.Users["projectEditor"]
	admin := m.Users["projectAdmin"]

	// assign twice -> still one edge (idempotent).
	_, err = repo.AssignScheduleItemUser(ctx, item.ID, editor)
	require.NoError(t, err)
	got, err := repo.AssignScheduleItemUser(ctx, item.ID, editor)
	require.NoError(t, err)
	assert.Len(t, got.Edges.Assignees, 1)

	// overbooking beyond cardinality=1 is allowed by design (Maximum Power).
	got, err = repo.AssignScheduleItemUser(ctx, item.ID, admin)
	require.NoError(t, err)
	assert.Len(t, got.Edges.Assignees, 2)

	// unassign; unassigning a non-assignee is a no-op, not an error.
	got, err = repo.UnassignScheduleItemUser(ctx, item.ID, editor)
	require.NoError(t, err)
	assert.Len(t, got.Edges.Assignees, 1)
	got, err = repo.UnassignScheduleItemUser(ctx, item.ID, editor)
	require.NoError(t, err)
	assert.Len(t, got.Edges.Assignees, 1)
}

// Shifts: nested on the block, hidden from the top-level list, "mine" matches
// via a claimed shift, and deleting the block cascades to its shifts.
func TestScheduleItemShifts(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)
	t0 := m.ReferenceNow

	block, err := repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Endurance", Start: t0, End: t0.Add(10 * time.Hour), ProjectID: m.Project,
	})
	require.NoError(t, err)
	shift1, err := repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Shift 1", Start: t0, End: t0.Add(90 * time.Minute), ProjectID: m.Project, ParentID: block.ID,
	})
	require.NoError(t, err)
	pause, err := repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Break", Start: t0.Add(90 * time.Minute), End: t0.Add(150 * time.Minute),
		ProjectID: m.Project, ParentID: block.ID, Kind: "break",
	})
	require.NoError(t, err)
	assert.Equal(t, "break", string(pause.Kind))

	// top-level list hides the shifts and nests them on the block, start-ordered.
	items, total, err := repo.GetScheduleItems(ctx, &repository.GetScheduleItemsParameters{
		ProjectID: m.Project, PaginationParameters: &repository.PaginationParameters{},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	require.Len(t, items[0].Edges.Shifts, 2)
	assert.Equal(t, shift1.ID, items[0].Edges.Shifts[0].ID)

	// "mine" finds the block through a claimed shift.
	editor := m.Users["projectEditor"]
	_, err = repo.AssignScheduleItemUser(ctx, shift1.ID, editor)
	require.NoError(t, err)
	items, total, err = repo.GetScheduleItems(ctx, &repository.GetScheduleItemsParameters{
		ProjectID: m.Project, AssigneeID: &editor, PaginationParameters: &repository.PaginationParameters{},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, block.ID, items[0].ID)

	// deleting the block cascades to the shifts.
	require.NoError(t, repo.DeleteScheduleItem(ctx, block.ID))
	_, err = repo.GetScheduleItem(ctx, shift1.ID)
	assert.Error(t, err, "shift must be gone with its block")
}

func TestScheduleItemTagProjectMismatch(t *testing.T) {
	ctx := context.Background()
	repo, m := seededRepo(t)

	other, err := repo.CreateProject(ctx, &repository.CreateProjectParameters{
		Name: "Other", Description: "d", Copyright: "c", CopyrightReference: "r",
		LocationName: "ln", LocationCode: "lc", LocationCity: "city",
	})
	require.NoError(t, err)
	foreignTag, err := repo.CreateImageTag(ctx, &repository.CreateImageTagParameters{
		Name: "foreign", Description: "d", Type: "manual", ProjectID: other.ID,
	})
	require.NoError(t, err)

	_, err = repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Sneaky", Start: m.ReferenceNow, End: m.ReferenceNow.Add(time.Hour),
		ProjectID: m.Project, TagIDs: []string{foreignTag.ID},
	})
	assert.ErrorIs(t, err, repository.ErrTagProjectMismatch, "cross-project tags rejected on create")

	item, err := repo.CreateScheduleItem(ctx, &repository.CreateScheduleItemParameters{
		Title: "Clean", Start: m.ReferenceNow, End: m.ReferenceNow.Add(time.Hour), ProjectID: m.Project,
	})
	require.NoError(t, err)
	_, err = repo.UpdateScheduleItem(ctx, item.ID, &repository.UpdateScheduleItemParameters{
		TagIDs: &[]string{foreignTag.ID},
	})
	assert.ErrorIs(t, err, repository.ErrTagProjectMismatch, "cross-project tags rejected on update")
}
