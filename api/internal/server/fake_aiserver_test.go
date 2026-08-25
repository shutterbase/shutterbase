package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/database"
	"github.com/shutterbase/shutterbase/internal/repository"
	"github.com/shutterbase/shutterbase/internal/seed"
	"github.com/shutterbase/shutterbase/pkg/aiserver"
)

func newFakeAIRemote(t *testing.T) (aiserver.Server, *seed.Manifest) {
	t.Helper()
	conn, err := database.NewConnection(&database.Options{DatabaseType: "sqlite", File: t.TempDir() + "/fakeai.db"})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	repo, err := repository.NewRepository(&repository.Options{DatabaseConnection: conn})
	require.NoError(t, err)
	m, err := seed.Seed(context.Background(), repo.Client, time.Now())
	require.NoError(t, err)
	return NewFakeAIRemote(repo.Client), m
}

func TestFakeAIFacesRoundTrip(t *testing.T) {
	remote, m := newFakeAIRemote(t)
	ctx := context.Background()
	imageID := m.Images[0]

	faces, err := remote.Faces(ctx, m.Project, imageID)
	require.NoError(t, err)
	require.NotEmpty(t, faces.Faces)

	for _, f := range faces.Faces {
		assert.Contains(t, f.PersonRef, "dev-person-")
		assert.GreaterOrEqual(t, f.X, 0.0)
		assert.LessOrEqual(t, f.X+f.W, 1.0)
		assert.GreaterOrEqual(t, f.Y, 0.0)
		assert.LessOrEqual(t, f.Y+f.H, 1.0)
	}

	ref := faces.Faces[0].PersonRef
	page, err := remote.PersonImages(ctx, m.Project, ref, 0, 100, false)
	require.NoError(t, err)
	require.Positive(t, page.Total)
	assert.Contains(t, refsOf(page.Items), imageID)
	cluster, ok := parseFakePersonRef(ref)
	require.True(t, ok)
	for _, item := range page.Items {
		assert.Equal(t, cluster, fakeCluster(item.ImageRef))
	}

	again, err := remote.Faces(ctx, m.Project, imageID)
	require.NoError(t, err)
	assert.Equal(t, faces, again)
}

func TestFakeAIPersonImagesPaging(t *testing.T) {
	remote, m := newFakeAIRemote(t)
	ctx := context.Background()

	first, err := remote.PersonImages(ctx, m.Project, "dev-person-0", 0, 1, false)
	require.NoError(t, err)
	collected := refsOf(first.Items)
	for page := 1; len(collected) < first.Total; page++ {
		p, err := remote.PersonImages(ctx, m.Project, "dev-person-0", page, 1, false)
		require.NoError(t, err)
		if len(p.Items) == 0 {
			break
		}
		collected = append(collected, p.Items[0].ImageRef)
	}
	assert.Len(t, collected, first.Total)
	require.NoError(t, unique(collected))

	beyond, err := remote.PersonImages(ctx, m.Project, "dev-person-0", 999, 20, false)
	require.NoError(t, err)
	assert.Empty(t, beyond.Items)
	assert.Equal(t, first.Total, beyond.Total)
}

func TestFakeAIPersonImagesUnknownRefs(t *testing.T) {
	remote, m := newFakeAIRemote(t)
	ctx := context.Background()

	for _, bad := range []string{"someone-else", "dev-person-42", "dev-person-x"} {
		resp, err := remote.PersonImages(ctx, m.Project, bad, 0, 20, false)
		require.NoError(t, err)
		assert.Empty(t, resp.Items)
		assert.Zero(t, resp.Total)
	}
}

func TestFakeAIFacesUnknownImage(t *testing.T) {
	remote, m := newFakeAIRemote(t)
	_, err := remote.Faces(context.Background(), m.Project, "does-not-exist")
	assert.True(t, errors.Is(err, aiserver.ErrNotFound))
}

func TestFakeAIPersonsRanked(t *testing.T) {
	remote, m := newFakeAIRemote(t)
	ctx := context.Background()

	expected := map[string]int{}
	for _, id := range m.Images {
		expected[fakePersonRef(fakeCluster(id))]++
	}

	persons, err := remote.Persons(ctx, []string{m.Project}, 0, 50)
	require.NoError(t, err)
	assert.Equal(t, len(expected), persons.Total)
	require.Len(t, persons.Items, persons.Total)

	prev := -1
	for _, entry := range persons.Items {
		count, ok := expected[entry.PersonRef]
		require.True(t, ok, "unexpected person %s", entry.PersonRef)
		assert.Equal(t, count, entry.Count)
		if prev >= 0 {
			assert.LessOrEqual(t, entry.Count, prev)
		}
		prev = entry.Count
		clusterIdx, ok := parseFakePersonRef(entry.PersonRef)
		require.True(t, ok)
		assert.Equal(t, clusterIdx, fakeCluster(entry.Sample.ImageRef))
	}
}

func TestFakeAIEmptyProxies(t *testing.T) {
	remote, m := newFakeAIRemote(t)
	ctx := context.Background()

	sim, err := remote.Similar(ctx, m.Project, m.Images[0], 0, 20)
	require.NoError(t, err)
	assert.Empty(t, sim.Items)

	candidates, err := remote.MergeCandidates(ctx, []string{m.Project}, 0, "")
	require.NoError(t, err)
	assert.Nil(t, candidates.Candidate)
	assert.Zero(t, candidates.Remaining)

	merges, err := remote.Merges(ctx, []string{m.Project})
	require.NoError(t, err)
	assert.Empty(t, merges.Items)

	require.NoError(t, remote.Prime(ctx, m.Project, aiserver.Project{}))
	require.NoError(t, remote.Recluster(ctx, m.Project))
	require.NoError(t, remote.DeleteImage(ctx, m.Project, m.Images[0]))
}

func refsOf(items []aiserver.PersonImage) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ImageRef)
	}
	return out
}

func unique(refs []string) error {
	seen := map[string]bool{}
	for _, r := range refs {
		if seen[r] {
			return errors.New("duplicate ref: " + r)
		}
		seen[r] = true
	}
	return nil
}
