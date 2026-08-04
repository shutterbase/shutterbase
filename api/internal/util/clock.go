package util

import (
	"sync"
	"time"
)

// Injectable server clock. Now() is the single source of "server now" for the
// WS time-tick and the time-relative dev quick-actions; it returns the live wall
// clock by default. The DEV /dev/clock quick-action freezes it to a fixed instant
// (and resets to live) so manual time-sync testing is deterministic.
//
// ponytail: a process-global guarded by one RWMutex — the laziest way to make a
// single logical clock injectable everywhere without threading a Clock through
// every constructor. It is only ever mutated by the DEV-gated /dev/clock route,
// so prod never leaves the live branch. Per-request clock injection is the
// upgrade path if simulated time ever needs to be scoped to a single request.
var (
	clockMu  sync.RWMutex
	frozenAt *time.Time
)

// Now returns the server clock: the frozen instant when a DEV freeze is active,
// else the live wall clock.
func Now() time.Time {
	clockMu.RLock()
	defer clockMu.RUnlock()
	if frozenAt != nil {
		return *frozenAt
	}
	return time.Now()
}

// FreezeClock pins Now() to at (DEV quick-action).
func FreezeClock(at time.Time) {
	clockMu.Lock()
	defer clockMu.Unlock()
	frozenAt = &at
}

// ResetClock returns Now() to the live wall clock (DEV quick-action).
func ResetClock() {
	clockMu.Lock()
	defer clockMu.Unlock()
	frozenAt = nil
}

// ClockFrozen reports the current server now and whether it is frozen.
func ClockFrozen() (time.Time, bool) {
	clockMu.RLock()
	defer clockMu.RUnlock()
	if frozenAt != nil {
		return *frozenAt, true
	}
	return time.Now(), false
}
