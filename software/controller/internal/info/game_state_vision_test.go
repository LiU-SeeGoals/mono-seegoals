package info

import "testing"

func TestTrackedRobotDoesNotReplaceNewerRawObservation(t *testing.T) {
	gs := NewGameState(10)
	gs.SetRobotFromVision(Blue, 1, Position{X: 200}, 1000, 100.020)
	gs.SetRobotFromTracked(Blue, 1, Position{X: 100}, 1001, 1000, 100.010)

	pos, err := gs.GetRobot(1, Blue).GetPosition()
	if err != nil || pos.X != 200 {
		t.Fatalf("older tracked frame replaced raw observation: pos=%v err=%v", pos, err)
	}

	gs.SetRobotFromTracked(Blue, 1, Position{X: 300}, 1002, 1000, 100.030)
	pos, err = gs.GetRobot(1, Blue).GetPosition()
	if err != nil || pos.X != 300 {
		t.Fatalf("newer tracked frame was not used: pos=%v err=%v", pos, err)
	}
}

func TestTrackedRobotFillsMissingRawObservation(t *testing.T) {
	gs := NewGameState(10)
	gs.SetRobotFromVision(Yellow, 1, Position{X: 200}, 1000, 100.020)
	gs.SetRobotFromTracked(Yellow, 2, Position{X: 300}, 1001, 1000, 100.010)

	pos, err := gs.GetRobot(2, Yellow).GetPosition()
	if err != nil || pos.X != 300 {
		t.Fatalf("tracked frame did not fill missing raw robot: pos=%v err=%v", pos, err)
	}
}

func TestRawRobotDoesNotReplaceNewerTrackedOrRawObservation(t *testing.T) {
	gs := NewGameState(10)
	gs.SetRobotFromTracked(Blue, 1, Position{X: 300}, 1000, 0, 100.030)
	gs.SetRobotFromVision(Blue, 1, Position{X: 200}, 1001, 100.020)
	gs.SetRobotFromVision(Blue, 1, Position{X: 400}, 1002, 100.040)
	gs.SetRobotFromVision(Blue, 1, Position{X: 350}, 1003, 100.035)

	pos, err := gs.GetRobot(1, Blue).GetPosition()
	if err != nil || pos.X != 400 {
		t.Fatalf("out-of-order raw frame replaced newer observation: pos=%v err=%v", pos, err)
	}
}
