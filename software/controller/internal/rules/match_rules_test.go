package rules

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

func eventAfter(commands ...info.RefCommand) *info.GameEvent {
	ge := info.NewGameEvent()
	for i, command := range commands {
		ge.UpdateFromRefCommand(command, uint64(100*(i+1)), 0, 0, info.UNINITIALIZED, 0, false)
	}
	return ge
}

func TestMatchRules(t *testing.T) {
	tests := []struct {
		name        string
		event       *info.GameEvent
		team        info.Team
		mayTouch    bool
		keepout     float64
		maxSpeed    float64
		defenseArea float64
	}{
		{"no referee", nil, info.Blue, true, 0, Unlimited, 0},
		{"halt", eventAfter(info.HALT), info.Blue, false, 0, 0, DefenseAreaStoppedClearance},
		{"stop", eventAfter(info.STOP), info.Blue, false, StopBallKeepout, ruleStopSpeed, DefenseAreaStoppedClearance},
		{"ball placement", eventAfter(info.BALL_PLACEMENT_BLUE), info.Blue, false, StopBallKeepout, ruleStopSpeed, DefenseAreaStoppedClearance},
		{"timeout", eventAfter(info.TIMEOUT_BLUE), info.Blue, false, 0, Unlimited, DefenseAreaStoppedClearance},
		{"our kickoff, preparing", eventAfter(info.PREPARE_KICKOFF_BLUE), info.Blue, false, 0, Unlimited, DefenseAreaStoppedClearance},
		{"our kickoff, started", eventAfter(info.PREPARE_KICKOFF_BLUE, info.NORMAL_START), info.Blue, true, 0, Unlimited, DefenseAreaStoppedClearance},
		{"their kickoff", eventAfter(info.PREPARE_KICKOFF_YELLOW, info.NORMAL_START), info.Blue, false, RestartBallKeepout, Unlimited, DefenseAreaStoppedClearance},
		{"our free kick", eventAfter(info.DIRECT_FREE_BLUE), info.Blue, true, 0, Unlimited, DefenseAreaStoppedClearance},
		{"their free kick", eventAfter(info.DIRECT_FREE_YELLOW), info.Blue, false, RestartBallKeepout, Unlimited, DefenseAreaStoppedClearance},
		{"our penalty, preparing", eventAfter(info.PREPARE_PENALTY_BLUE), info.Blue, false, 0, Unlimited, DefenseAreaStoppedClearance},
		{"force start", eventAfter(info.FORCE_START), info.Blue, true, 0, Unlimited, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := FromEvent(tt.event, tt.team)
			if r.MayTouchBall != tt.mayTouch || r.BallKeepout != tt.keepout ||
				r.MaxSpeed != tt.maxSpeed || r.DefenseAreaClearance != tt.defenseArea {
				t.Fatalf("got touch=%v keepout=%v speed=%v area=%v, want %v %v %v %v",
					r.MayTouchBall, r.BallKeepout, r.MaxSpeed, r.DefenseAreaClearance,
					tt.mayTouch, tt.keepout, tt.maxSpeed, tt.defenseArea)
			}
		})
	}
}

func TestTakenRestartIsOpenPlay(t *testing.T) {
	ge := eventAfter(info.DIRECT_FREE_YELLOW)
	ge.SetBallMoved()
	r := FromEvent(ge, info.Blue)
	if !r.BallInPlay || !r.MayTouchBall || r.BallKeepout != 0 {
		t.Fatalf("taken free kick: %+v", r)
	}
}
