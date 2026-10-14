package info

import "time"

type TrackedBall struct {
	Pos       Position
	Vel       Position
	Timestamp float64
	Valid     bool
	updatedAt int64
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
	if !tb.Valid || now.UnixMilli()-tb.updatedAt > maxAge.Milliseconds() {
		return Position{}, false
	}
	return tb.Vel, true
}
