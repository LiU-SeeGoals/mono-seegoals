package info

import "testing"

func TestHoldLastKnownBallRefreshesRecentPosition(t *testing.T) {
	gs := NewGameState(4)
	gs.SetBall(100, 200, 0, 1000)

	if !gs.HoldLastKnownBallFor(1200, 500) {
		t.Fatal("HoldLastKnownBallFor returned false for a recent ball")
	}

	pos, ts, err := gs.Ball.GetPositionTime()
	if err != nil {
		t.Fatal(err)
	}
	if ts != 1200 {
		t.Fatalf("ball timestamp = %d, want 1200", ts)
	}
	if pos.X != 100 || pos.Y != 200 {
		t.Fatalf("ball position = %v, want last known position", pos)
	}
}

func TestHoldLastKnownBallDoesNotRefreshOldPosition(t *testing.T) {
	gs := NewGameState(4)
	gs.SetBall(100, 200, 0, 1000)

	if gs.HoldLastKnownBallFor(2000, 500) {
		t.Fatal("HoldLastKnownBallFor returned true for an old ball")
	}

	_, ts, err := gs.Ball.GetPositionTime()
	if err != nil {
		t.Fatal(err)
	}
	if ts != 1000 {
		t.Fatalf("ball timestamp = %d, want original timestamp", ts)
	}
}

func TestHoldMissingRobotPositionsRefreshesRecentMissingRobot(t *testing.T) {
	gs := NewGameState(4)
	gs.SetYellowRobot(1, 300, 400, 0.5, 1000)

	seen := [TEAM_SIZE]bool{}
	gs.HoldMissingRobotPositionsFor(Yellow, seen, 1200, 500)

	pos, ts, err := gs.GetRobot(1, Yellow).GetPositionTime()
	if err != nil {
		t.Fatal(err)
	}
	if ts != 1200 {
		t.Fatalf("robot timestamp = %d, want 1200", ts)
	}
	if pos.X != 300 || pos.Y != 400 || pos.Angle != 0.5 {
		t.Fatalf("robot position = %v, want last known position", pos)
	}
}

func TestHoldMissingRobotPositionsDoesNotRefreshSeenRobot(t *testing.T) {
	gs := NewGameState(4)
	gs.SetYellowRobot(1, 300, 400, 0.5, 1000)

	seen := [TEAM_SIZE]bool{}
	seen[1] = true
	gs.HoldMissingRobotPositionsFor(Yellow, seen, 1200, 500)

	_, ts, err := gs.GetRobot(1, Yellow).GetPositionTime()
	if err != nil {
		t.Fatal(err)
	}
	if ts != 1000 {
		t.Fatalf("robot timestamp = %d, want original timestamp", ts)
	}
}

func TestHoldTrackedBallUsesLastKnownPositionWithZeroVelocity(t *testing.T) {
	gs := NewGameState(4)
	pos := Position{X: 100, Y: 200, Z: 0, Angle: 0}
	gs.SetTrackedBall(pos, Position{X: 1, Y: 2, Z: 0, Angle: 0}, 1.0)

	if !gs.HoldTrackedBall(1.1) {
		t.Fatal("HoldTrackedBall returned false for a valid tracked ball")
	}

	heldPos, ok := gs.GetTrackedBall().GetTrackedPosition()
	if !ok {
		t.Fatal("tracked ball is invalid")
	}
	heldVel, ok := gs.GetTrackedBall().GetTrackedVelocity()
	if !ok {
		t.Fatal("tracked ball velocity is invalid")
	}
	if heldPos != pos {
		t.Fatalf("tracked ball position = %v, want %v", heldPos, pos)
	}
	if heldVel != (Position{}) {
		t.Fatalf("tracked ball velocity = %v, want zero", heldVel)
	}
}
