package info

import "testing"

func TestNewGameEventHasNoAnnouncedNextCommand(t *testing.T) {
	if got := NewGameEvent().NextCommand; got != UNINITIALIZED {
		t.Fatalf("new game event next command = %s, want uninitialized", got)
	}
}

func TestNextCommandSurvivesAbsentRepeatPacketDuringStop(t *testing.T) {
	gameEvent := NewGameEvent()
	gameEvent.UpdateFromRefCommand(STOP, 100, 0, 0, DIRECT_FREE_BLUE, 0, false)
	gameEvent.UpdateFromRefCommand(STOP, 100, 0, 0, UNINITIALIZED, 0, false)

	if gameEvent.NextCommand != DIRECT_FREE_BLUE {
		t.Fatalf("next command = %s, want preserved direct free blue", gameEvent.NextCommand)
	}
}

func TestNextCommandClearsForNewStopWithoutAnnouncement(t *testing.T) {
	gameEvent := NewGameEvent()
	gameEvent.UpdateFromRefCommand(STOP, 100, 0, 0, DIRECT_FREE_BLUE, 0, false)
	gameEvent.UpdateFromRefCommand(STOP, 200, 0, 0, UNINITIALIZED, 0, false)

	if gameEvent.NextCommand != UNINITIALIZED {
		t.Fatalf("next command = %s, want uninitialized for a new STOP", gameEvent.NextCommand)
	}
}

func TestNextCommandSurvivesAbsentRepeatPacketDuringBallPlacement(t *testing.T) {
	gameEvent := NewGameEvent()
	gameEvent.UpdateFromRefCommand(BALL_PLACEMENT_YELLOW, 100, 0, 0, DIRECT_FREE_YELLOW, 0, false)
	gameEvent.UpdateFromRefCommand(BALL_PLACEMENT_YELLOW, 100, 0, 0, UNINITIALIZED, 0, false)

	if gameEvent.NextCommand != DIRECT_FREE_YELLOW {
		t.Fatalf("next command = %s, want preserved direct free yellow", gameEvent.NextCommand)
	}
}
