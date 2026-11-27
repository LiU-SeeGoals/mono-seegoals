package client

import (
	"math"
	"testing"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func linearSpeed(t *testing.T, m *SimMotion, a action.Action) float64 {
	t.Helper()
	cmd, ok := m.Command(a)
	if !ok {
		t.Fatalf("%T not supported", a)
	}
	v := cmd.GetMoveCommand().GetLocalVelocity()
	return math.Hypot(float64(v.GetForward()), float64(v.GetLeft()))
}

func TestSimMotionAppliesRequestedMinimumLinearSpeed(t *testing.T) {
	m := NewSimMotion(DefaultSimMotionLimits)
	got := linearSpeed(t, m, &action.MoveTo{Dest: info.Position{X: 50}, MinLinearSpeed: 0.5})
	if math.Abs(got-0.5) > 1e-6 {
		t.Fatalf("speed = %.6f, want 0.5 m/s floor", got)
	}
}

func TestSimMotionCapsMinimumLinearSpeedAtMaximum(t *testing.T) {
	m := NewSimMotion(DefaultSimMotionLimits)
	got := linearSpeed(t, m, &action.MoveTo{Dest: info.Position{X: 50}, MinLinearSpeed: 2})
	if math.Abs(got-1) > 1e-6 {
		t.Fatalf("speed = %.6f, want the 1 m/s maximum", got)
	}
}

func TestSimMotionRespectsConfiguredMaxSpeed(t *testing.T) {
	m := NewSimMotion(SimMotionLimits{MaxLinearMPS: 2.5, MaxAngularRadPS: 3})
	got := linearSpeed(t, m, &action.MoveTo{Dest: info.Position{X: 5000}})
	if math.Abs(got-2.5) > 1e-6 {
		t.Fatalf("speed = %.6f, want 2.5 m/s", got)
	}
}

func TestSimMotionLimitsAcceleration(t *testing.T) {
	clock := info.NewManualClock(time.Unix(0, 0))
	m := NewSimMotion(SimMotionLimits{MaxLinearMPS: 2, MaxAngularRadPS: 3, MaxLinearAccMPS2: 3})
	m.now = clock.Now
	far := &action.MoveTo{Id: 1, Dest: info.Position{X: 5000}}

	m.Command(&action.Stop{Id: 1}) // start at rest
	clock.Advance(100 * time.Millisecond)
	if got := linearSpeed(t, m, far); math.Abs(got-0.3) > 1e-6 {
		t.Fatalf("after 100 ms speed = %.4f, want 0.3 m/s (3 m/s² · 0.1 s)", got)
	}
	clock.Advance(100 * time.Millisecond)
	if got := linearSpeed(t, m, far); math.Abs(got-0.6) > 1e-6 {
		t.Fatalf("after 200 ms speed = %.4f, want 0.6 m/s", got)
	}
	// A long gap (robot unseen) does not keep throttling from stale state.
	clock.Advance(time.Second)
	if got := linearSpeed(t, m, far); math.Abs(got-2) > 1e-6 {
		t.Fatalf("after a stale gap speed = %.4f, want 2 m/s", got)
	}
}

func TestSimMotionWithoutAccelerationLimitJumpsImmediately(t *testing.T) {
	m := NewSimMotion(DefaultSimMotionLimits)
	m.Command(&action.Stop{Id: 1})
	if got := linearSpeed(t, m, &action.MoveTo{Id: 1, Dest: info.Position{X: 5000}}); math.Abs(got-1) > 1e-6 {
		t.Fatalf("speed = %.4f, want 1 m/s", got)
	}
}
