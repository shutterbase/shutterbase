package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/shutterbase/shutterbase/ent"
	"github.com/shutterbase/shutterbase/ent/upload"
)

// The two pure pieces of the upload metrics: the active-time heuristic and the
// review-cycle accounting. Both are exercised without a database.

func TestFoldTaggingActivity(t *testing.T) {
	idle := 2 * time.Minute
	t0 := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)

	// first action only anchors the clock
	seconds, last, changed := foldTaggingActivity(0, nil, t0, idle)
	assert.True(t, changed)
	assert.Equal(t, 0, seconds)
	assert.Equal(t, t0, last)

	// a short gap is working time
	seconds, last, _ = foldTaggingActivity(seconds, &last, t0.Add(30*time.Second), idle)
	assert.Equal(t, 30, seconds)

	// a long gap is a break and contributes nothing
	seconds, last, _ = foldTaggingActivity(seconds, &last, t0.Add(90*time.Minute), idle)
	assert.Equal(t, 30, seconds)

	// work resumes and accumulates again
	seconds, last, _ = foldTaggingActivity(seconds, &last, t0.Add(91*time.Minute), idle)
	assert.Equal(t, 90, seconds)

	// exactly at the threshold still counts
	seconds, last, _ = foldTaggingActivity(seconds, &last, t0.Add(93*time.Minute), idle)
	assert.Equal(t, 210, seconds)

	// an out-of-order action is ignored, clock included
	before := last
	seconds, last, changed = foldTaggingActivity(seconds, &last, t0.Add(time.Minute), idle)
	assert.False(t, changed)
	assert.Equal(t, 210, seconds)
	assert.Equal(t, before, last)
}

func TestApplyCycle(t *testing.T) {
	created := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	item := &ent.Upload{State: upload.StateOpen, CreatedAt: created}

	// first submit: banks creation -> ready and counts one cycle
	seconds, cycles, start := applyCycle(item, upload.StateReady, created.Add(time.Hour))
	assert.Equal(t, 3600, seconds)
	assert.Equal(t, 1, cycles)
	assert.Nil(t, start, "no cycle is running while the upload sits with the reviewer")

	// sent back: a new cycle starts, nothing banked yet
	item = &ent.Upload{State: upload.StateReady, CreatedAt: created, TimeToReadySeconds: seconds, ReviewCycles: cycles}
	sentBack := created.Add(2 * time.Hour)
	seconds, cycles, start = applyCycle(item, upload.StateOpen, sentBack)
	assert.Equal(t, 3600, seconds)
	assert.Equal(t, 1, cycles)
	assert.Equal(t, sentBack, *start)

	// resubmitted: the rework time is added on top and a second cycle counted
	item = &ent.Upload{State: upload.StateOpen, CreatedAt: created, TimeToReadySeconds: seconds, ReviewCycles: cycles, CycleStartedAt: start}
	seconds, cycles, start = applyCycle(item, upload.StateReady, sentBack.Add(10*time.Minute))
	assert.Equal(t, 4200, seconds)
	assert.Equal(t, 2, cycles)
	assert.Nil(t, start)

	// accepting a submitted upload changes no cycle accounting
	item = &ent.Upload{State: upload.StateReady, CreatedAt: created, TimeToReadySeconds: seconds, ReviewCycles: cycles}
	seconds, cycles, start = applyCycle(item, upload.StateReviewed, sentBack.Add(time.Hour))
	assert.Equal(t, 4200, seconds)
	assert.Equal(t, 2, cycles)
	assert.Nil(t, start)

	// a reviewer accepting straight out of "open" still closes the cycle
	item = &ent.Upload{State: upload.StateOpen, CreatedAt: created}
	seconds, cycles, _ = applyCycle(item, upload.StateReviewed, created.Add(time.Minute))
	assert.Equal(t, 60, seconds)
	assert.Equal(t, 1, cycles)
}
