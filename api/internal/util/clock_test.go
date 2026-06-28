package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The injectable clock: default live, freeze pins Now(), reset returns to live.
func TestClockFreezeReset(t *testing.T) {
	t.Cleanup(ResetClock)

	_, frozen := ClockFrozen()
	assert.False(t, frozen, "clock is live by default")

	at := time.Date(2031, 5, 6, 7, 8, 9, 0, time.UTC)
	FreezeClock(at)
	got, frozen := ClockFrozen()
	assert.True(t, frozen)
	assert.True(t, got.Equal(at))
	assert.True(t, Now().Equal(at), "Now() returns the frozen instant")

	ResetClock()
	_, frozen = ClockFrozen()
	assert.False(t, frozen)
	assert.WithinDuration(t, time.Now(), Now(), time.Second, "Now() is live again")
}
