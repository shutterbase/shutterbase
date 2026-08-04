package util

import "time"

// TimeOffsetFreshness is the window within which a camera's time offset counts
// as up to date. Beyond it, the create-upload page disables the camera
// (outdatedTimeOffsetFound). Mirrors the UI's dateTimeUtil.timeOffsetUpToDate.
const TimeOffsetFreshness = 24 * time.Hour

// TimeOffsetUpToDate reports whether a time offset recorded at serverTime is
// still fresh relative to now (serverTime > now - 24h).
func TimeOffsetUpToDate(serverTime, now time.Time) bool {
	return serverTime.After(now.Add(-TimeOffsetFreshness))
}
