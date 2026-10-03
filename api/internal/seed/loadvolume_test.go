package seed

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The volume line is printed BEFORE the run so an operator can judge a 250k seed
// while it is still a plan. Both of its counts were wrong, in opposite directions,
// and neither was visible: the assignments figure omitted the two calendar tags
// every photo carries (so a 250k run reported 40% fewer rows than it writes), and
// the jsonb figure reported one rebuild per ASSIGNMENT while the loaders call
// rebuildImageTagsJSON once per IMAGE — overstating the read-modify-write work by
// three to six times. An estimate nobody can trust stops being read at all.

// volumeNames is a run of requested photo names, as the loaders build them.
func volumeNames(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("FSG_LW%05d.jpg", i)
	}
	return names
}

// presentIn returns the `existing` map the loaders pass in: the names already in the
// database, keyed by name.
func presentIn(names []string) map[string]string {
	out := make(map[string]string, len(names))
	for _, name := range names {
		out[name] = "img" + name
	}
	return out
}

func TestEstimateLoadVolumeCountsTheCalendarPair(t *testing.T) {
	const photos = 250_000
	names := volumeNames(photos)

	// Default + drawn extras + the calendar pair. The pair is the part that was
	// missing: every photo carries it, it is not counted against --tag-count, and an
	// estimate that leaves it out understates by two rows per photo at any setting.
	for _, tc := range []struct {
		extras       int
		wantPerPhoto int
	}{
		{0, 5}, // unpinned: 1 + the split's 1.9 rounded up + 2
		{1, 4},
		{2, 5},
		{3, 6},
	} {
		vol := estimateLoadVolume(names, nil, tc.extras)
		require.Equal(t, photos, vol.PhotosToCreate)
		assert.Equal(t, photos*tc.wantPerPhoto, vol.TagAssignments,
			"--tag-count %d: every photo writes Default, %d drawn and %d calendar rows",
			tc.extras, tc.wantPerPhoto-1-calendarTagsPerPhoto(), calendarTagsPerPhoto())
	}
}

func TestEstimateLoadVolumeCountsOneRebuildPerPhoto(t *testing.T) {
	const photos = 250_000
	names := volumeNames(photos)

	for _, extras := range []int{0, 1, 2, 3} {
		vol := estimateLoadVolume(names, nil, extras)
		assert.Equal(t, photos, vol.JSONBRebuilds,
			"--tag-count %d: createTagAssignments rebuilds each image's jsonb once, whatever it wrote", extras)
		assert.Less(t, vol.JSONBRebuilds, vol.TagAssignments,
			"--tag-count %d: one rebuild per assignment would be %d rebuilds, not %d",
			extras, vol.TagAssignments, vol.JSONBRebuilds)
	}
}

// The unpinned run is the default one, so its average is the number most operators
// ever see: 1.9 drawn extras, not the 1.8 the comment used to quote (and not the 3
// the line used to report, which read as "three random tags per photo").
func TestEstimateLoadVolumeAveragesTheSplitItself(t *testing.T) {
	vol := estimateLoadVolume(volumeNames(1000), nil, 0)
	// A literal, not 1000*assignmentsPerPhoto(0) — that would move with any change
	// to the function under test and prove nothing.
	assert.Equal(t, 1000*(1+2+calendarTagsPerPhoto()), vol.TagAssignments,
		"the 30/50/20 split averages 1.9 extras per photo: 0.3x1 + 0.5x2 + 0.2x3")
	assert.Equal(t, 1000, vol.JSONBRebuilds)
}

// A re-run reports the work LEFT, not the size of the final set: that is the whole
// point of the line, so a re-run must not inflate either count.
func TestEstimateLoadVolumeExcludesPhotosAlreadyPresent(t *testing.T) {
	names := volumeNames(1000)
	existing := presentIn(names[:600])

	vol := estimateLoadVolume(names, existing, 2)
	assert.Equal(t, 400, vol.PhotosToCreate)
	assert.Equal(t, 600, vol.AlreadyPresent)
	assert.Equal(t, 400*(1+2+calendarTagsPerPhoto()), vol.TagAssignments)
	assert.Equal(t, 400, vol.JSONBRebuilds)

	// An identical re-run writes nothing, and says so with zeros rather than the
	// full request.
	done := estimateLoadVolume(names, presentIn(names), 2)
	assert.Equal(t, 0, done.PhotosToCreate)
	assert.Zero(t, done.TagAssignments)
	assert.Zero(t, done.JSONBRebuilds)
	assert.Equal(t, len(names), done.AlreadyPresent)
}

// The one place the number can drift is calendarTagsPerPhoto, which is why it is
// derived by asking the function that decides the pair instead of writing 2 in a
// comment. A photo resolves two names, and the count follows.
func TestCalendarTagsPerPhotoFollowsTheCalendarTags(t *testing.T) {
	assert.Equal(t, 2, calendarTagsPerPhoto(),
		"a photo carries its date and its weekday — nothing else is unconditional")
	assert.Equal(t, 1+1+2, assignmentsPerPhoto(1))
	assert.Equal(t, 1+3+2, assignmentsPerPhoto(3))
	assert.Equal(t, 1+2+2, assignmentsPerPhoto(0),
		"unpinned, the split averages 1.9 drawn extras and rounds to 2")
}
