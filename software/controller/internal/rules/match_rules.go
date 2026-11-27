package rules

import (
	"math"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

// Unlimited is MatchRules.MaxSpeed when the rules set no speed limit.
var Unlimited = math.Inf(1)

// MatchRules is what the rules allow one team in the current referee state.
// It is a value derived from info.GameEvent: compute it with For and read it;
// nothing writes back into the referee state.
type MatchRules struct {
	Team  info.Team
	State info.RefState

	// Halted: no robot may move.
	Halted bool
	// Stopped: STOP or ball placement, both teams keep StopBallKeepout and
	// drive at most ruleStopSpeed.
	Stopped bool
	// BallInPlay: the ball is in play (normal play, or a restart was taken).
	BallInPlay bool
	// OurRestart: this team takes the pending kickoff, free kick or penalty.
	OurRestart bool
	// MayTouchBall: a robot of this team may touch the ball now.
	MayTouchBall bool
	// BallKeepout is the centre distance every robot of this team must keep
	// from the ball, or 0 when there is none.
	BallKeepout float64
	// MaxSpeed is the speed limit in m/s (0 while halted, Unlimited when none).
	MaxSpeed float64
	// DefenseAreaClearance is the minimum centre distance from either
	// defense area (0 when only the normal motion clearance applies).
	DefenseAreaClearance float64
}

// For returns the rules for team in gi's current referee state. A nil or
// incomplete gi yields open play.
func For(gi *info.GameInfo, team info.Team) MatchRules {
	if gi == nil || gi.Status == nil {
		return FromEvent(nil, team)
	}
	return FromEvent(gi.Status.GetGameEvent(), team)
}

// FromEvent derives the rules for team from a referee state.
func FromEvent(ge *info.GameEvent, team info.Team) MatchRules {
	r := MatchRules{Team: team, State: info.STATE_PLAYING, BallInPlay: true, MaxSpeed: Unlimited, MayTouchBall: true}
	if ge == nil {
		return r
	}

	r.State = ge.CurrentState
	r.BallInPlay = ge.BallInPlay
	restartIssued := ge.RefCommand == info.NORMAL_START // kickoff / penalty may be taken

	switch ge.CurrentState {
	case info.STATE_HALTED:
		r.Halted = true
		r.MaxSpeed = 0
		r.MayTouchBall = false
	case info.STATE_TIMEOUT:
		r.MayTouchBall = false
	case info.STATE_STOPPED, info.STATE_BALL_PLACEMENT:
		// We never place the ball ourselves, so the placing team gets no
		// exception here.
		r.Stopped = true
		r.MaxSpeed = ruleStopSpeed
		r.MayTouchBall = false
		r.BallKeepout = StopBallKeepout
	case info.STATE_FREE_KICK:
		r.OurRestart = ge.TeamWithPossession == team
		r.MayTouchBall = r.OurRestart
		if !r.OurRestart {
			r.BallKeepout = RestartBallKeepout
		}
	case info.STATE_KICKOFF_PREPARATION:
		r.OurRestart = ge.TeamWithPossession == team
		r.MayTouchBall = r.OurRestart && restartIssued
		if !r.OurRestart {
			r.BallKeepout = RestartBallKeepout
		}
	case info.STATE_PENALTY_PREPARATION:
		// Penalty formations (1 m behind the ball) are placed by the referee
		// states; there is no circular keep-out.
		r.OurRestart = ge.TeamWithPossession == team
		r.MayTouchBall = r.OurRestart && restartIssued
	}

	if r.BallInPlay {
		// A taken restart is open play even before the state machine catches up.
		r.MayTouchBall = r.State != info.STATE_HALTED
		r.BallKeepout = 0
	} else {
		r.DefenseAreaClearance = DefenseAreaStoppedClearance
	}
	return r
}
