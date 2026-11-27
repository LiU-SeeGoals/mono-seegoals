package info

import (
	"testing"
	"time"
)

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

func TestRepeatPacketKeepsPreviousState(t *testing.T) {
	ge := NewGameEvent()
	ge.UpdateFromRefCommand(STOP, 100, 0, 0, UNINITIALIZED, 0, false)
	ge.UpdateFromRefCommand(PREPARE_KICKOFF_BLUE, 200, 0, 0, UNINITIALIZED, 0, false)
	ge.UpdateFromRefCommand(PREPARE_KICKOFF_BLUE, 200, 0, 0, UNINITIALIZED, 0, false)
	if ge.PreviousState != STATE_STOPPED {
		t.Fatalf("previous state = %s after a repeat packet, want Stopped", ge.PreviousState)
	}
}

func TestObserveRestartPutsBallInPlayAfterItMoved(t *testing.T) {
	now := time.Unix(100, 0)
	ge := NewGameEvent()
	ge.UpdateFromRefCommand(DIRECT_FREE_BLUE, 100, 0, 0, UNINITIALIZED, 10_000_000, true)

	ge.ObserveRestart(Position{X: 1000}, true, now, DivisionB)
	ge.ObserveRestart(Position{X: 1000 + BallInPlayDistance - 1}, true, now, DivisionB)
	if ge.BallInPlay {
		t.Fatal("ball in play before it moved BallInPlayDistance")
	}
	ge.ObserveRestart(Position{X: 1000 + BallInPlayDistance}, true, now, DivisionB)
	if !ge.BallInPlay || !ge.RestartKicked || ge.CurrentState != STATE_PLAYING {
		t.Fatal("moved ball did not put the free kick in play")
	}
	if anchor, ok := ge.RestartBallAnchor(); !ok || anchor.X != 1000 {
		t.Fatalf("anchor = %v %v, want x=1000", anchor, ok)
	}
}

func TestObserveRestartWaitsForNormalStart(t *testing.T) {
	now := time.Unix(100, 0)
	ge := NewGameEvent()
	ge.UpdateFromRefCommand(PREPARE_KICKOFF_BLUE, 100, 0, 0, UNINITIALIZED, 0, false)
	ge.ObserveRestart(Position{}, true, now, DivisionB)
	ge.ObserveRestart(Position{X: 500}, true, now.Add(time.Minute), DivisionB)
	if ge.BallInPlay || ge.CurrentState != STATE_KICKOFF_PREPARATION {
		t.Fatal("kickoff preparation went into play before NORMAL_START")
	}
	ge.UpdateFromRefCommand(NORMAL_START, 200, 0, 0, UNINITIALIZED, 0, false)
	ge.ObserveRestart(Position{X: 500}, true, now, DivisionB)
	ge.ObserveRestart(Position{X: 600}, true, now, DivisionB)
	if !ge.BallInPlay {
		t.Fatal("kicked kickoff not in play")
	}
}

func TestObserveRestartFallbackTimeLimit(t *testing.T) {
	start := time.Unix(100, 0)
	tests := []struct {
		name    string
		prepare RefCommand
		start   RefCommand
		want    RefState
	}{
		{"free kick continues as play", STOP, DIRECT_FREE_YELLOW, STATE_PLAYING},
		{"kickoff continues as play", PREPARE_KICKOFF_YELLOW, NORMAL_START, STATE_PLAYING},
		{"penalty returns to stop", PREPARE_PENALTY_YELLOW, NORMAL_START, STATE_STOPPED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge := NewGameEvent()
			ge.UpdateFromRefCommand(tt.prepare, 100, 0, 0, UNINITIALIZED, 0, false)
			ge.UpdateFromRefCommand(tt.start, 200, 0, 0, UNINITIALIZED, 0, false)
			limit := RestartTimeLimit(ge.CurrentState, DivisionB)
			ge.ObserveRestart(Position{}, true, start, DivisionB)
			ge.ObserveRestart(Position{}, true, start.Add(limit-time.Millisecond), DivisionB)
			if ge.CurrentState == tt.want {
				t.Fatal("restart ended before its time limit")
			}
			ge.ObserveRestart(Position{}, true, start.Add(limit), DivisionB)
			if ge.CurrentState != tt.want || ge.RestartKicked {
				t.Fatalf("state = %s kicked=%v, want %s and not kicked", ge.CurrentState, ge.RestartKicked, tt.want)
			}
		})
	}
}
