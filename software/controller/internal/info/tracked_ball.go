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

func (tb *TrackedBall) SetTracked(pos Position, vel Position, ts float64) {
	tb.Pos = pos
	tb.Vel = vel
	tb.Timestamp = ts
	tb.Valid = true
	tb.updatedAt = time.Now().UnixMilli()
}

func (tb *TrackedBall) GetTrackedPosition() (Position, bool) {
	if !tb.Valid {
		return Position{}, false
	}
	return tb.Pos, true
}

func (tb *TrackedBall) GetTrackedVelocity() (Position, bool) {
	if !tb.Valid {
		return Position{}, false
	}
	return tb.Vel, true
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
