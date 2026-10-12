package info

import "testing"

func TestProjectTrajectoryToXProjectsLatestBallDirection(t *testing.T) {
	ball := NewBall(4)
	ball.SetPositionTime(0, 0, 0, 1000)
	ball.SetPositionTime(1000, 200, 0, 1020)

	hit, ok := ball.ProjectTrajectoryToX(3000)
	if !ok {
		t.Fatal("expected ball trajectory to cross X=3000")
	}
	if hit.X != 3000 || hit.Y != 600 {
		t.Fatalf("trajectory hit = %+v, want X=3000 Y=600", hit)
	}
}

func TestProjectTrajectoryToXRejectsPointBehindRay(t *testing.T) {
	ball := NewBall(4)
	ball.SetPositionTime(0, 0, 0, 1000)
	ball.SetPositionTime(1000, 200, 0, 1020)

	if hit, ok := ball.ProjectTrajectoryToX(-1000); ok {
		t.Fatalf("trajectory unexpectedly projected backward to %+v", hit)
	}
}

func TestProjectTrajectoryToXRejectsStaleSamples(t *testing.T) {
	ball := NewBall(4)
	ball.SetPositionTime(0, 0, 0, 1000)
	ball.SetPositionTime(1000, 200, 0, 1000+BallTrajectoryMaxSampleMillis+1)

	if hit, ok := ball.ProjectTrajectoryToX(3000); ok {
		t.Fatalf("stale trajectory unexpectedly projected to %+v", hit)
	}
}

func TestProjectTrajectoryToXRejectsJitter(t *testing.T) {
	ball := NewBall(4)
	ball.SetPositionTime(0, 0, 0, 1000)
	ball.SetPositionTime(BallTrajectoryMinDisplacement/2, 0, 0, 1020)

	if hit, ok := ball.ProjectTrajectoryToX(3000); ok {
		t.Fatalf("ball jitter unexpectedly projected to %+v", hit)
	}
}
