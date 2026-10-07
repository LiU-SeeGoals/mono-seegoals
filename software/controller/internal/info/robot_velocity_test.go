package info

import (
	"math"
	"testing"
)

func TestRobotVelocityIgnoresSamplesWithSameTimestamp(t *testing.T) {
	robot := NewRobot(1, Blue, 10)
	robot.SetPositionTime(0, 0, 0, 1000)
	robot.SetPositionTime(10, 0, 0, 1010)
	robot.SetPositionTime(12, 0, 0, 1010)

	vel := robot.GetVelocity()
	if math.IsNaN(vel.X) || math.IsInf(vel.X, 0) || math.IsNaN(vel.Y) || math.IsInf(vel.Y, 0) {
		t.Fatalf("velocity (%v, %v) is not finite", vel.X, vel.Y)
	}
	if math.Abs(vel.X-1.2) > 1e-9 {
		t.Fatalf("velocity %.3f m/s, want 1.2", vel.X)
	}
}
