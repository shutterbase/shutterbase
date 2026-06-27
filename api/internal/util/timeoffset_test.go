package util_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/shutterbase/shutterbase/internal/util"
)

// Seed unit: the 24h freshness rule (used to disable cameras on the upload page).
// Mirrors the seed's fresh (serverTime=now) vs deliberately-stale (now-25h) offsets.
func TestTimeOffsetUpToDate(t *testing.T) {
	now := time.Now()
	assert.True(t, util.TimeOffsetUpToDate(now, now), "serverTime=now is fresh")
	assert.True(t, util.TimeOffsetUpToDate(now.Add(-23*time.Hour), now), "23h old is still fresh")
	assert.False(t, util.TimeOffsetUpToDate(now.Add(-25*time.Hour), now), "25h old is stale")
}
