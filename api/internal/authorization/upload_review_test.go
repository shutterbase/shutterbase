package authorization

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/imagetag"
	"github.com/shutterbase/shutterbase/ent/upload"
	"github.com/shutterbase/shutterbase/ent/user"
)

func up(owner *ent.User, state upload.State) *ent.Upload {
	return &ent.Upload{ProjectID: proj, UserID: owner.ID, State: state}
}

func tag(name string, t imagetag.Type) *ent.ImageTag {
	return &ent.ImageTag{Name: name, Type: t, ProjectID: proj}
}

func TestCanTransitionUpload(t *testing.T) {
	owner := usr(user.RoleUser, pa(proj, RoleProjectEditor))
	reviewer := usr(user.RoleUser, pa(proj, RoleProjectAdmin))
	stranger := usr(user.RoleUser, pa(proj, RoleProjectEditor))

	// the photographer submits, and that is all they may do
	assert.True(t, CanTransitionUpload(owner, up(owner, upload.StateOpen), upload.StateReady))
	assert.False(t, CanTransitionUpload(owner, up(owner, upload.StateOpen), upload.StateReviewed))
	assert.False(t, CanTransitionUpload(owner, up(owner, upload.StateReady), upload.StateOpen),
		"a submitted upload is the reviewer's; the photographer cannot pull it back")
	assert.False(t, CanTransitionUpload(stranger, up(owner, upload.StateOpen), upload.StateReady))

	// the reviewer sends back, accepts and reopens
	assert.True(t, CanTransitionUpload(reviewer, up(owner, upload.StateReady), upload.StateOpen))
	assert.True(t, CanTransitionUpload(reviewer, up(owner, upload.StateReady), upload.StateReviewed))
	assert.True(t, CanTransitionUpload(reviewer, up(owner, upload.StateReviewed), upload.StateOpen))

	assert.False(t, CanTransitionUpload(owner, nil, upload.StateReady))
}

func TestCanAssignTag(t *testing.T) {
	owner := usr(user.RoleUser, pa(proj, RoleProjectEditor))
	reviewer := usr(user.RoleUser, pa(proj, RoleProjectAdmin))
	viewer := usr(user.RoleUser, pa(proj, RoleProjectViewer))
	img := &ent.Image{ProjectID: proj, UserID: owner.ID}

	official := tag("Race", imagetag.TypeManual)
	custom := tag("todo", imagetag.TypeCustom)
	errTag := tag(ReviewErrorTagName, imagetag.TypeCustom)
	rejectedTag := tag(ReviewRejectedTagName, imagetag.TypeCustom)

	open := up(owner, upload.StateOpen)
	ready := up(owner, upload.StateReady)
	reviewed := up(owner, upload.StateReviewed)

	// review flow off -> unchanged §4.5 behavior
	assert.True(t, CanAssignTag(owner, img, ready, official, false))
	assert.False(t, CanAssignTag(viewer, img, open, custom, false), "projectViewer never assigns")

	// review flow on, upload still open -> the photographer tags freely
	assert.True(t, CanAssignTag(owner, img, open, official, true))
	assert.True(t, CanAssignTag(owner, img, open, custom, true))

	// submitted -> official tags freeze, custom tags stay editable
	for _, submitted := range []*ent.Upload{ready, reviewed} {
		assert.False(t, CanAssignTag(owner, img, submitted, official, true))
		assert.True(t, CanAssignTag(owner, img, submitted, custom, true))
	}

	// the reserved review tags are the reviewer's alone, in every state
	for _, reserved := range []*ent.ImageTag{errTag, rejectedTag} {
		for _, state := range []*ent.Upload{open, ready, reviewed} {
			assert.False(t, CanAssignTag(owner, img, state, reserved, true), reserved.Name)
			assert.True(t, CanAssignTag(reviewer, img, state, reserved, true), reserved.Name)
		}
	}
	assert.True(t, IsReviewErrorTag("Error"), "the reserved name is case-insensitive")
	assert.True(t, IsReviewRejectedTag("Rejected"))
	assert.True(t, IsReviewerOnlyTag("ERROR"))
	assert.True(t, IsReviewerOnlyTag("rejected"))
	assert.False(t, IsReviewerOnlyTag("rejects"))

	// the reviewer is never frozen out
	assert.True(t, CanAssignTag(reviewer, img, reviewed, official, true))
}

func TestCanAddImagesToUpload(t *testing.T) {
	owner := usr(user.RoleUser, pa(proj, RoleProjectEditor))
	reviewer := usr(user.RoleUser, pa(proj, RoleProjectAdmin))

	assert.True(t, CanAddImagesToUpload(owner, up(owner, upload.StateOpen), true))
	assert.False(t, CanAddImagesToUpload(owner, up(owner, upload.StateReady), true))
	assert.True(t, CanAddImagesToUpload(owner, up(owner, upload.StateReady), false),
		"without the review flow the state is meaningless")
	assert.True(t, CanAddImagesToUpload(reviewer, up(owner, upload.StateReady), true))
}
