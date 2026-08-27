package authorization

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/ent/user"
)

func TestReservedTagNamespaceIsCaseInsensitive(t *testing.T) {
	for _, n := range []string{"public", "Public", "PUBLIC", "internal", "Error", "REJECTED"} {
		assert.True(t, IsReservedTag(n), n)
	}
	assert.False(t, IsReservedTag("publicity"))
	assert.False(t, IsReservedTag("Race"))
	assert.True(t, IsPublicTag("Public"))
	assert.False(t, IsPublicTag("internal"))
}

// The public tag decides what the world sees: projectAdmin-only whatever the
// review mode, while every other rule of CanAssignTag stays as it was.
func TestCanAssignPublicTagIsProjectAdminOnly(t *testing.T) {
	editor := usr(user.RoleUser, pa(proj, RoleProjectEditor))
	projectAdmin := usr(user.RoleUser, pa(proj, RoleProjectAdmin))
	admin := usr(user.RoleAdmin)
	img := &ent.Image{ProjectID: proj, UserID: editor.ID}
	public := tag("public", imagetag.TypeCustom)
	publicUpper := tag("Public", imagetag.TypeCustom)
	custom := tag("todo", imagetag.TypeCustom)
	open := up(editor, upload.StateOpen)

	for _, reviewEnabled := range []bool{false, true} {
		assert.False(t, CanAssignTag(editor, img, open, public, reviewEnabled), "editor, review=%v", reviewEnabled)
		assert.False(t, CanAssignTag(editor, img, open, publicUpper, reviewEnabled), "case must not bypass")
		assert.True(t, CanAssignTag(projectAdmin, img, open, public, reviewEnabled))
		assert.True(t, CanAssignTag(admin, img, open, public, reviewEnabled))
		// unrelated custom tags keep their §4.5 behaviour
		assert.True(t, CanAssignTag(editor, img, open, custom, reviewEnabled))
	}
}

func TestCanManageReservedTag(t *testing.T) {
	assert.False(t, CanManageReservedTag(usr(user.RoleUser, pa(proj, RoleProjectEditor)), proj))
	assert.True(t, CanManageReservedTag(usr(user.RoleUser, pa(proj, RoleProjectAdmin)), proj))
	assert.True(t, CanManageReservedTag(usr(user.RoleAdmin), proj))
	assert.False(t, CanManageGalleries(usr(user.RoleUser, pa(proj, RoleProjectAdmin))))
	assert.True(t, CanManageGalleries(usr(user.RoleAdmin)))
}
