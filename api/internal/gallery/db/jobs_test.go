package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The in-memory store mirrors the Postgres semantics the worker relies on:
// gallery scoping, lease ownership, retry accounting, reuse by hash.
func TestMemoryJobsLeaseSemantics(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryJobs()
	require.NoError(t, m.Create(ctx, &Job{ID: "a", GalleryKey: "fsg", ProjectID: "p", FilterHash: "h1", Manifest: []string{"i1"}}))
	require.NoError(t, m.Create(ctx, &Job{ID: "b", GalleryKey: "fsa", ProjectID: "p", FilterHash: "h1", Manifest: []string{"i1"}}))

	_, err := m.Get(ctx, "fsa", "a")
	assert.ErrorIs(t, err, ErrJobNotFound, "jobs are gallery-scoped")

	r, err := m.Reusable(ctx, "fsg", "h1", time.Now())
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "a", r.ID)

	j, err := m.Claim(ctx, "fsg", "tok1", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, j)
	assert.Equal(t, "a", j.ID)
	assert.Equal(t, 1, j.Attempts)
	none, err := m.Claim(ctx, "fsg", "tok2", time.Minute)
	require.NoError(t, err)
	assert.Nil(t, none, "a leased job is not claimable")

	assert.Error(t, m.Heartbeat(ctx, "a", "wrong", time.Minute), "lease token guards every write")
	require.NoError(t, m.Heartbeat(ctx, "a", "tok1", time.Minute))

	// first failure requeues, second fails for good
	require.NoError(t, m.Fail(ctx, "a", "tok1", "boom"))
	j, _ = m.Get(ctx, "fsg", "a")
	assert.Equal(t, JobQueued, j.Status)
	j, err = m.Claim(ctx, "fsg", "tok3", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, j)
	require.NoError(t, m.Fail(ctx, "a", "tok3", "boom again"))
	j, _ = m.Get(ctx, "fsg", "a")
	assert.Equal(t, JobFailed, j.Status)
	none, _ = m.Claim(ctx, "fsg", "tok4", time.Minute)
	assert.Nil(t, none, "attempts exhausted")

	// the other gallery's job is untouched and finishes normally
	j, err = m.Claim(ctx, "fsa", "tok5", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, j)
	require.NoError(t, m.Finish(ctx, "b", "tok5", 1, 123, "zips/b.zip", time.Now().Add(time.Hour)))
	r, _ = m.Reusable(ctx, "fsa", "h1", time.Now())
	require.NotNil(t, r)
	assert.Equal(t, JobDone, r.Status)
	exp, _ := m.Expired(ctx, "fsa", time.Now().Add(2*time.Hour))
	assert.Len(t, exp, 1)
	n, _ := m.CountActive(ctx, "fsg")
	assert.Equal(t, 0, n)
}
