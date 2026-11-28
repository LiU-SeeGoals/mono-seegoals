package info

import "time"

type TrackedBall struct {
	Pos       Position
	Vel       Position
	Timestamp float64
	Valid     bool
	updatedAt int64
}

type TrackedBallState struct {
	Position  Position
	Velocity  Position
	Timestamp float64
	Age       time.Duration
}

func NewTrackedBall() *TrackedBall {
	return &TrackedBall{}
}

// TrackedStaleAfter is how long a tracker sample stays usable when the
// tracker stops sending.
const TrackedStaleAfter = 500 * time.Millisecond

// SetTracked stores a tracker sample received at now (GameInfo.Now, so
// ManualClock tests and replays age samples correctly).
func (tb *TrackedBall) SetTracked(pos Position, vel Position, ts float64, now time.Time) {
	tb.Pos = pos
	tb.Vel = vel
	tb.Timestamp = ts
	tb.Valid = true
	tb.updatedAt = now.UnixMilli()
}

// GetTrackedPosition returns the last tracked position however old it is.
// Prefer GetFreshTrackedPosition.
func (tb *TrackedBall) GetTrackedPosition() (Position, bool) {
	if !tb.Valid {
		return Position{}, false
	}
	return tb.Pos, true
}

// GetFreshTrackedPosition returns the tracked position if it is at most
// maxAge old at now.
func (tb *TrackedBall) GetFreshTrackedPosition(now time.Time, maxAge time.Duration) (Position, bool) {
	state, ok := tb.GetFreshBallState(now, maxAge)
	return state.Position, ok
}

func (tb *TrackedBall) GetFreshTrackedVelocity(now time.Time, maxAge time.Duration) (Position, bool) {
	state, ok := tb.GetFreshBallState(now, maxAge)
	if !ok {
		return Position{}, false
	}

	return state.Velocity, true
}

func (tb *TrackedBall) GetFreshBallState(now time.Time, maxAge time.Duration) (TrackedBallState, bool) {
	if !tb.Valid || tb.updatedAt == 0 {
		return TrackedBallState{}, false
	}

	age := time.Duration(now.UnixMilli()-tb.updatedAt) * time.Millisecond
	if age < 0 || age > maxAge {
		return TrackedBallState{}, false
	}

	return TrackedBallState{
		Position:  tb.Pos,
		Velocity:  tb.Vel,
		Timestamp: tb.Timestamp,
		Age:       age,
	}, true
}
