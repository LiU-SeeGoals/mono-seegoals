package info

import (
	"math"
	"sync"
	"time"
)

const (
	// TrackedBallFreshness is the maximum age of SSL-Vision tracker data used
	// for prediction. Older data is ignored instead of extrapolated forever.
	TrackedBallFreshness = 250 * time.Millisecond
	metersToMillimeters  = 1000.0
)

type TrackedBall struct {
	mu        sync.RWMutex
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
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.Pos = pos
	tb.Vel = vel
	tb.Timestamp = ts
	tb.Valid = true
	tb.updatedAt = time.Now().UnixMilli()
}

func (tb *TrackedBall) GetTrackedPosition() (Position, bool) {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	if !tb.Valid {
		return Position{}, false
	}
	return tb.Pos, true
}

func (tb *TrackedBall) GetTrackedVelocity() (Position, bool) {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	if !tb.Valid {
		return Position{}, false
	}
	return tb.Vel, true
}

func (tb *TrackedBall) GetFreshTrackedVelocity(now time.Time, maxAge time.Duration) (Position, bool) {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	if !tb.Valid || now.UnixMilli()-tb.updatedAt > maxAge.Milliseconds() {
		return Position{}, false
	}
	return tb.Vel, true
}

func (tb *TrackedBall) freshState(now time.Time) (Position, Position, bool) {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	if !tb.Valid || now.UnixMilli()-tb.updatedAt > TrackedBallFreshness.Milliseconds() {
		return Position{}, Position{}, false
	}
	return tb.Pos, tb.Vel, true
}

// PredictPosition extrapolates the tracked SSL-Vision ball state. Tracked
// positions are millimetres while tracked velocities are metres per second;
// the conversion intentionally lives here rather than in tactical plans.
func (tb *TrackedBall) PredictPosition(after time.Duration, minSpeedMetersPerSecond float64) (Position, bool) {
	pos, vel, ok := tb.freshState(time.Now())
	if !ok || vel.Norm2d() <= minSpeedMetersPerSecond {
		return Position{}, false
	}

	seconds := after.Seconds()
	pos.X += vel.X * metersToMillimeters * seconds
	pos.Y += vel.Y * metersToMillimeters * seconds
	pos.Z += vel.Z * metersToMillimeters * seconds
	return pos, true
}

// ProjectTrajectoryToX returns where the fresh SSL-Vision tracked velocity ray
// crosses targetX. It only projects forward along the measured direction.
func (tb *TrackedBall) ProjectTrajectoryToX(targetX float64) (Position, bool) {
	pos, vel, ok := tb.freshState(time.Now())
	if !ok || math.Abs(vel.X) < 1e-9 {
		return Position{}, false
	}

	velocityMM := vel.Scale(metersToMillimeters)
	minimumSpeed := BallTrajectoryMinDisplacement /
		(float64(BallTrajectoryMaxSampleMillis) / float64(time.Second/time.Millisecond))
	if velocityMM.Norm2d() < minimumSpeed {
		return Position{}, false
	}

	seconds := (targetX - pos.X) / velocityMM.X
	if seconds < 0 {
		return Position{}, false
	}

	return Position{
		X: targetX,
		Y: pos.Y + velocityMM.Y*seconds,
		Z: pos.Z + velocityMM.Z*seconds,
	}, true
}
