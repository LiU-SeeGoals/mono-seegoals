package referee

import (
	"sync"
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestRefereePreparationAndTimeoutTransitions(t *testing.T) {
	tests := []struct {
		name   string
		events []state_machine.EventName
		want   state_machine.StateName
	}{
		{"cancel kickoff", []state_machine.EventName{STOP, PREPARE_KICKOFF, STOP}, "STOP"},
		{"enter timeout", []state_machine.EventName{STOP, TIMEOUT}, "TIMEOUT"},
		{"exit timeout", []state_machine.EventName{STOP, TIMEOUT, STOP}, "STOP"},
		{"halt timeout", []state_machine.EventName{STOP, TIMEOUT, HALT}, "HALT"},
		{"start during timeout", []state_machine.EventName{TIMEOUT}, "TIMEOUT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gi := info.NewGameInfo(10)
			var activities [info.TEAM_SIZE]activity.Activity
			handler := &ai.ActivityHandler{Activities: &activities, Activity_lock: &sync.Mutex{}}
			ref := NewRefereeHandler(gi, nil, info.Blue, handler)
			for _, event := range tt.events {
				ref.refereeSM.TriggerEvent(event)
			}
			if got := ref.refereeSM.CurrentStateName(); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func newTestHandler(gi *info.GameInfo, team info.Team, robots []info.ID) *RefereeHandler {
	var activities [info.TEAM_SIZE]activity.Activity
	handler := &ai.ActivityHandler{Activities: &activities, Activity_lock: &sync.Mutex{}}
	return NewRefereeHandler(gi, robots, team, handler)
}

func TestPenaltyTimeoutReturnsToStop(t *testing.T) {
	gi := info.NewGameInfo(10)
	gi.State.SetBall(0, 0, 0, 1)
	ge := gi.Status.GetGameEvent()
	ref := newTestHandler(gi, info.Blue, nil)

	ge.UpdateFromRefCommand(info.PREPARE_PENALTY_BLUE, 100, 0, 0, info.UNINITIALIZED, 10_000_000, true)
	ref.HandleReferee()
	ge.UpdateFromRefCommand(info.NORMAL_START, 200, 0, 0, info.UNINITIALIZED, 0, true)
	gi.ObserveRestart()
	ref.HandleReferee()
	if ref.refereeSM.CurrentStateName() != StateStop || ge.CurrentState != info.STATE_STOPPED || ge.BallInPlay {
		t.Fatalf("expired penalty: referee %s, game state %s, in play %v; want STOP, Stopped, false",
			ref.refereeSM.CurrentStateName(), ge.CurrentState, ge.BallInPlay)
	}
}

func TestStopIsLeftToThePlanner(t *testing.T) {
	gi := info.NewGameInfo(10)
	ref := newTestHandler(gi, info.Blue, []info.ID{1, 2})
	gi.Status.GetGameEvent().UpdateFromRefCommand(info.STOP, 100, 0, 0, info.UNINITIALIZED, 0, false)
	if ref.HandleReferee() {
		t.Fatal("STOP handled by the referee; the planner should own it")
	}
	gi.Status.GetGameEvent().UpdateFromRefCommand(info.BALL_PLACEMENT_YELLOW, 200, 0, 0, info.UNINITIALIZED, 0, false)
	if !ref.HandleReferee() {
		t.Fatal("ball placement not handled by the referee")
	}
}

// Both teams' handlers read the same GameEvent; neither may change it.
func TestHandlersDoNotWriteRefereeState(t *testing.T) {
	gi := info.NewGameInfo(10)
	gi.State.SetBall(0, 0, 0, 1)
	gi.State.SetBlueRobot(1, -1000, 0, 0, 1)
	gi.State.SetYellowRobot(1, 1000, 0, 0, 1)
	ge := gi.Status.GetGameEvent()
	blue := newTestHandler(gi, info.Blue, []info.ID{1})
	yellow := newTestHandler(gi, info.Yellow, []info.ID{1})

	for i, command := range []info.RefCommand{info.STOP, info.DIRECT_FREE_BLUE, info.PREPARE_PENALTY_YELLOW, info.NORMAL_START} {
		ge.UpdateFromRefCommand(command, uint64(100*(i+1)), 0, 0, info.UNINITIALIZED, 10_000_000, true)
		gi.ObserveRestart()
		before := *ge
		blue.HandleReferee()
		yellow.HandleReferee()
		if *ge != before {
			t.Fatalf("%v: a referee handler changed the shared GameEvent", command)
		}
	}
}

func TestFreeKickHandsOverOnceTheBallMoved(t *testing.T) {
	gi := info.NewGameInfo(10)
	gi.State.SetBall(0, 0, 0, 1)
	ge := gi.Status.GetGameEvent()
	ref := newTestHandler(gi, info.Blue, nil)

	ge.UpdateFromRefCommand(info.DIRECT_FREE_BLUE, 100, 0, 0, info.UNINITIALIZED, 10_000_000, true)
	gi.ObserveRestart()
	ref.HandleReferee()
	if ref.refereeSM.CurrentStateName() != StateFreeKick || ge.BallInPlay {
		t.Fatal("free kick did not start")
	}

	gi.State.SetBall(info.BallInPlayDistance+1, 0, 0, 2)
	gi.ObserveRestart()
	ref.HandleReferee()
	if !ge.BallInPlay || !ge.RestartKicked || !ref.freeKick.kickTaken {
		t.Fatal("moved ball did not put the free kick in play")
	}
}

func TestFreeKickStartsWithoutStopAndRepeatedPacketDoesNotRestartIt(t *testing.T) {
	gi := info.NewGameInfo(10)
	gi.State.SetBall(0, 0, 0, 1)
	ref := newTestHandler(gi, info.Blue, nil)
	ge := gi.Status.GetGameEvent()
	ge.UpdateFromRefCommand(info.FORCE_START, 100, 0, 0, info.UNINITIALIZED, 0, false)
	ref.HandleReferee()
	ge.UpdateFromRefCommand(info.DIRECT_FREE_BLUE, 200, 0, 0, info.UNINITIALIZED, 10_000_000, true)
	ref.HandleReferee()
	if ref.refereeSM.CurrentStateName() != "FREEKICK" {
		t.Fatal("free kick ignored after RUNNING without an intermediate STOP")
	}
	started := ref.freeKick.freeKickStart
	ref.HandleReferee()
	if ref.freeKick.freeKickStart != started {
		t.Fatal("repeat command reset free-kick initialization")
	}
	ge.SetBallMoved()
	ref.refereeSM.TriggerEvent(GAME_RUNNING_DETECTED)
	ref.HandleReferee()
	if ref.refereeSM.CurrentStateName() != "RUNNING" {
		t.Fatal("old free-kick command restarted a completed restart")
	}
	ge.UpdateFromRefCommand(info.DIRECT_FREE_BLUE, 300, 0, 0, info.UNINITIALIZED, 10_000_000, true)
	ref.HandleReferee()
	if ref.refereeSM.CurrentStateName() != "FREEKICK" {
		t.Fatal("new command of the same type was ignored")
	}
}
