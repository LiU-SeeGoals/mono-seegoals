package info

type TrackedBall struct {
	Pos       Position
	Vel       Position
	Timestamp float64
	Valid     bool
}

func NewTrackedBall() *TrackedBall {
	return &TrackedBall{}
}

func (tb *TrackedBall) SetTracked(pos Position, vel Position, ts float64) {
	tb.Pos = pos
	tb.Vel = vel
	tb.Timestamp = ts
	tb.Valid = true
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

