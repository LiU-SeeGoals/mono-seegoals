package info

import (
	"math"
	"testing"
)

func TestRobotAngularVelocityAcrossPi(t *testing.T) {
	r := NewRobot(1, Blue, 10)
	// Turning counter-clockwise through +π at 0.002 rad/ms.
	r.SetPositionTime(0, 0, math.Pi-0.01, 0)
	r.SetPositionTime(0, 0, -math.Pi+0.01, 10)

	got := r.GetVelocity().Angle
	if math.Abs(got-0.002) > 1e-9 {
		t.Fatalf("angular velocity = %g rad/ms, want 0.002", got)
	}
}

func TestRobotVelocityLinear(t *testing.T) {
	r := NewRobot(1, Blue, 10)
	r.SetPositionTime(0, 0, 0, 0)
	r.SetPositionTime(10, -20, 0, 10)
	r.SetPositionTime(20, -40, 0, 20)
	v := r.GetVelocity()
	if math.Abs(v.X-1) > 1e-9 || math.Abs(v.Y+2) > 1e-9 {
		t.Fatalf("velocity = (%g, %g), want (1, -2) mm/ms", v.X, v.Y)
	}
}

func TestRobotAccelerationUniform(t *testing.T) {
	r := NewRobot(1, Blue, 10)
	// x = 0.5·a·t² with a = 0.001 mm/ms² (1 m/s²), sampled every 10 ms.
	const a = 0.001
	for i := 0; i <= 5; i++ {
		tm := float64(i * 10)
		r.SetPositionTime(0.5*a*tm*tm, 0, 0, int64(tm))
	}
	if got := r.GetAcceleration(); math.Abs(got-a) > 1e-9 {
		t.Fatalf("acceleration = %g mm/ms², want %g", got, a)
	}
}

func TestRobotAccelerationIgnoresDuplicateTimestamps(t *testing.T) {
	r := NewRobot(1, Blue, 10)
	r.SetPositionTime(0, 0, 0, 0)
	r.SetPositionTime(5, 0, 0, 0)
	r.SetPositionTime(10, 0, 0, 10)
	if got := r.GetAcceleration(); math.IsNaN(got) || math.IsInf(got, 0) {
		t.Fatalf("acceleration = %g, want a finite value", got)
	}
}

func TestVec2AndPose2D(t *testing.T) {
	v := Vec2{3, 4}
	if v.Len() != 5 || v.Unit().Len() != 1 {
		t.Fatalf("Len/Unit wrong: %v %v", v.Len(), v.Unit())
	}
	if (Vec2{}).Unit() != (Vec2{}) {
		t.Fatal("zero vector Unit should stay zero")
	}
	r := Vec2{1, 0}.Rotated(math.Pi / 2)
	if math.Abs(r.X) > 1e-12 || math.Abs(r.Y-1) > 1e-12 {
		t.Fatalf("Rotated = %v, want (0, 1)", r)
	}
	pose := Position{X: 1, Y: 2, Z: 9, Angle: 3}.Pose()
	if pose.Position() != (Position{X: 1, Y: 2, Angle: 3}) {
		t.Fatalf("Pose round trip = %v", pose.Position())
	}
	if e := (Pose2D{Heading: math.Pi - 0.1}).HeadingError(-math.Pi + 0.1); math.Abs(e-0.2) > 1e-12 {
		t.Fatalf("HeadingError = %g, want 0.2", e)
	}
}
