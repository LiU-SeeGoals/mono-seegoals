package info

import "testing"

func TestUnseenRobotExpiresWhileVisionContinues(t *testing.T) {
	gs := NewGameState(10)
	gs.SetBlueRobot(1, 0, 0, 0, 1000)
	gs.SetBlueRobot(2, 100, 0, 0, 1000)

	// Robot 2 keeps being seen; robot 1 was removed from the field.
	gs.SetBlueRobot(2, 100, 0, 0, 1000+RobotUnseenAfterMs+1)
	gs.SetMessageReceivedTime(1000 + RobotUnseenAfterMs + 1)
	gs.Update()

	if gs.Blue_team[1].IsActive() {
		t.Fatal("robot unseen for longer than RobotUnseenAfterMs is still active")
	}
	if !gs.Blue_team[2].IsActive() {
		t.Fatal("robot seen this frame was deactivated")
	}

	gs.SetBlueRobot(1, 0, 0, 0, 1000+RobotUnseenAfterMs+50)
	if !gs.Blue_team[1].IsActive() {
		t.Fatal("robot was not reactivated by a new sample")
	}
}

func TestRecentlySeenRobotStaysActive(t *testing.T) {
	gs := NewGameState(10)
	gs.SetYellowRobot(3, 0, 0, 0, 1000)
	gs.SetMessageReceivedTime(1000 + RobotUnseenAfterMs)
	gs.Update()
	if !gs.Yellow_team[3].IsActive() {
		t.Fatal("robot within the unseen window was deactivated")
	}
}
