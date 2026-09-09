package info

import "testing"

func TestRepeatedFreeKickPacketPreservesBallInPlay(t *testing.T) {
	ge := NewGameEvent()
	ge.UpdateFromRefCommand(DIRECT_FREE_BLUE, 100, 0, 0, UNINITIALIZED, 10_000_000, true)
	ge.SetBallMoved()
	ge.UpdateFromRefCommand(DIRECT_FREE_BLUE, 100, 0, 0, UNINITIALIZED, 9_000_000, true)
	if !ge.BallInPlay || ge.CurrentState != STATE_PLAYING {
		t.Fatal("repeat packet reset a completed restart")
	}
	ge.UpdateFromRefCommand(DIRECT_FREE_BLUE, 200, 0, 0, UNINITIALIZED, 10_000_000, true)
	if ge.BallInPlay || ge.CurrentState != STATE_FREE_KICK {
		t.Fatal("new free kick did not start a fresh restart")
	}
}

func TestPreparationTimeoutDoesNotStartPlay(t *testing.T) {
	for _, command := range []RefCommand{PREPARE_KICKOFF_BLUE, PREPARE_PENALTY_YELLOW} {
		ge := NewGameEvent()
		ge.UpdateFromRefCommand(command, 100, 0, 0, UNINITIALIZED, 0, true)
		if ge.BallInPlay || ge.CurrentState == STATE_PLAYING {
			t.Fatalf("%v timer incorrectly started play", command)
		}
	}
}

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
