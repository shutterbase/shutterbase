package authorization

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/ent/user"
)

func TestCanManageScheduleItem(t *testing.T) {
	assert.True(t, CanManageScheduleItem(usr(user.RoleAdmin), proj), "global admin")
	assert.True(t, CanManageScheduleItem(usr(user.RoleUser, pa(proj, RoleProjectAdmin)), proj), "projectAdmin of this project")
	assert.False(t, CanManageScheduleItem(usr(user.RoleUser, pa("other", RoleProjectAdmin)), proj), "projectAdmin elsewhere")
	assert.False(t, CanManageScheduleItem(usr(user.RoleUser, pa(proj, RoleProjectEditor)), proj), "editor cannot manage the pool")
	assert.False(t, CanManageScheduleItem(nil, proj))
}

func TestCanManageScheduleAssignment(t *testing.T) {
	editor := usr(user.RoleUser, pa(proj, RoleProjectEditor))
	viewer := usr(user.RoleUser, pa(proj, RoleProjectViewer))
	padmin := usr(user.RoleUser, pa(proj, RoleProjectAdmin))
	other := uuid.New()

	assert.True(t, CanManageScheduleAssignment(editor, proj, editor.ID), "editor manages self")
	assert.False(t, CanManageScheduleAssignment(editor, proj, other), "editor cannot manage others")
	assert.False(t, CanManageScheduleAssignment(viewer, proj, viewer.ID), "viewer cannot cover events")
	assert.True(t, CanManageScheduleAssignment(padmin, proj, other), "projectAdmin manages anyone")
	assert.True(t, CanManageScheduleAssignment(usr(user.RoleAdmin), proj, other), "global admin manages anyone")

	inactive := usr(user.RoleUser, pa(proj, RoleProjectEditor))
	inactive.Active = false
	assert.False(t, CanManageScheduleAssignment(inactive, proj, inactive.ID), "deactivated user has no schedule rights")
}

func TestCanApplyUploadTimeline(t *testing.T) {
	owner := usr(user.RoleUser, pa(proj, RoleProjectEditor))
	padmin := usr(user.RoleUser, pa(proj, RoleProjectAdmin))
	stranger := usr(user.RoleUser, pa("other", RoleProjectEditor))
	up := func(state upload.State) *ent.Upload {
		return &ent.Upload{ProjectID: proj, UserID: owner.ID, State: state}
	}

	// Review flow off: the owner may apply in any state.
	assert.True(t, CanApplyUploadTimeline(owner, up(upload.StateOpen), false))
	assert.True(t, CanApplyUploadTimeline(owner, up(upload.StateReady), false))

	// Review flow on: official tags freeze once the upload left open — only the
	// reviewer may still re-apply.
	assert.True(t, CanApplyUploadTimeline(owner, up(upload.StateOpen), true))
	assert.False(t, CanApplyUploadTimeline(owner, up(upload.StateReady), true))
	assert.False(t, CanApplyUploadTimeline(owner, up(upload.StateReviewed), true))
	assert.True(t, CanApplyUploadTimeline(padmin, up(upload.StateReady), true))

	// Not the owner, not a reviewer -> never.
	assert.False(t, CanApplyUploadTimeline(stranger, up(upload.StateOpen), false))
	assert.False(t, CanApplyUploadTimeline(nil, up(upload.StateOpen), false))
}
