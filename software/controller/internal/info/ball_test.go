package info

import (
	"testing"
	"time"
)

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

func TestGameStateProjectsFreshTrackedBallFromVision(t *testing.T) {
	state := NewGameState(4)
	state.SetBall(0, 0, 0, 1000)
	state.SetBall(1000, 200, 0, 1020)
	state.SetTrackedBall(
		Position{X: 1000, Y: 100},
		Position{X: 1, Y: 0.5},
		1,
	)

	hit, ok := state.ProjectBallTrajectoryToX(3000)
	if !ok {
		t.Fatal("expected fresh tracked trajectory to cross X=3000")
	}
	if hit.X != 3000 || hit.Y != 1100 {
		t.Fatalf("tracked trajectory hit = %+v, want X=3000 Y=1100", hit)
	}
}

func TestGameStateFallsBackToRawVisionTrajectoryWhenTrackIsStale(t *testing.T) {
	state := NewGameState(4)
	state.SetBall(0, 0, 0, 1000)
	state.SetBall(1000, 200, 0, 1020)
	state.SetTrackedBall(Position{X: 1000}, Position{X: 1, Y: 1}, 1)
	state.TrackedBall.mu.Lock()
	state.TrackedBall.updatedAt = time.Now().Add(-TrackedBallFreshness - time.Millisecond).UnixMilli()
	state.TrackedBall.mu.Unlock()

	hit, ok := state.ProjectBallTrajectoryToX(3000)
	if !ok {
		t.Fatal("expected raw vision fallback trajectory to cross X=3000")
	}
	if hit.X != 3000 || hit.Y != 600 {
		t.Fatalf("raw fallback hit = %+v, want X=3000 Y=600", hit)
	}
}

func TestBallPositionAfterUsesTrackedVisionUnits(t *testing.T) {
	state := NewGameState(4)
	state.SetTrackedBall(
		Position{X: 1000, Y: 200},
		Position{X: 2, Y: -1},
		1,
	)

	predicted, ok := state.BallPositionAfter(500*time.Millisecond, 0.3)
	if !ok {
		t.Fatal("expected tracked ball prediction")
	}
	if predicted.X != 2000 || predicted.Y != -300 {
		t.Fatalf("predicted position = %+v, want X=2000 Y=-300", predicted)
	}
}
