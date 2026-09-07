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
