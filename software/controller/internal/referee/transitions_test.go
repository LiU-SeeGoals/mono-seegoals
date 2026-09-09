package referee

import (
	"sync"
	"testing"

	coreai "github.com/LiU-SeeGoals/controller/internal/ai"
	activity "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestRefereePreparationAndTimeoutTransitions(t *testing.T) {
	tests := []struct {
		name   string
		events []EventName
		want   StateName
	}{
		{"cancel kickoff", []EventName{STOP, PREPARE_KICKOFF, STOP}, "STOP"},
		{"enter timeout", []EventName{STOP, TIMEOUT}, "TIMEOUT"},
		{"exit timeout", []EventName{STOP, TIMEOUT, STOP}, "STOP"},
		{"halt timeout", []EventName{STOP, TIMEOUT, HALT}, "HALT"},
		{"start during timeout", []EventName{TIMEOUT}, "TIMEOUT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gi := info.NewGameInfo(10)
			var activities [info.TEAM_SIZE]activity.Activity
			handler := &coreai.ActivityHandler{Activities: &activities, Activity_lock: &sync.Mutex{}}
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

func TestPenaltyTimeoutReturnsToStop(t *testing.T) {
	gi := info.NewGameInfo(10)
	ge := gi.Status.GetGameEvent()
	ge.CurrentState = info.STATE_PENALTY_PREPARATION
	ge.CurrentActionTimeRemainingValid = true
	ge.CurrentActionTimeRemaining = 0
	var activities [info.TEAM_SIZE]activity.Activity
	handler := &coreai.ActivityHandler{Activities: &activities, Activity_lock: &sync.Mutex{}}
	ref := NewRefereeHandler(gi, nil, info.Blue, handler)
	for _, event := range []EventName{STOP, PREPARE_PENALTY, NORMAL_START} {
		ref.refereeSM.TriggerEvent(event)
	}
	ref.refereeSM.Update()
	if ref.refereeSM.CurrentStateName() != "STOP" || ge.CurrentState != info.STATE_STOPPED || ge.BallInPlay {
		t.Fatal("expired penalty did not return both referee representations to STOP")
	}
}

func TestFreeKickStartsWithoutStopAndRepeatedPacketDoesNotRestartIt(t *testing.T) {
	gi := info.NewGameInfo(10)
	gi.State.SetBall(0, 0, 0, 1)
	var activities [info.TEAM_SIZE]activity.Activity
	handler := &coreai.ActivityHandler{Activities: &activities, Activity_lock: &sync.Mutex{}}
	ref := NewRefereeHandler(gi, nil, info.Blue, handler)
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
