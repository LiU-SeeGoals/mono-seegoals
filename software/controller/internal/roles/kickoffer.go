package roles

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	KICKOFFER_GOTO_POS = 0
	KICKOFFER_PASS_BALL  = 1
	KICKOFFER_SHOOT_GOAL = 2
	KICKOFFER_STAY       = 0
)

type KickofferRole struct {
	*RobotRole
}

func NewKickofferRole(robotID ID, gameScenario any) *KickofferRole {
	return &KickofferRole{
		RobotRole: NewRobotRole(robotID, "KICKoff", gameScenario),
	}
}

func (r *KickofferRole) GetStateMachineName() string {
	if r.isBallOwner {
		switch r.stateMachine {
		case KICKOFFER_GOTO_POS:
			return "GOTO_POS"
		case KICKOFFER_PASS_BALL:
			return "PASS_BALL"
		}
	} else {
		switch r.stateMachine {
		case KICKOFFER_STAY:
			return "STAY"
		}
	}
	return fmt.Sprintf("SM_%d", r.stateMachine)
}

func (r *KickofferRole) StateMachine(gi GameInfo, team Team, g GameScenarioInterface, receiverID ID) ai.Activity {
	var activity ai.Activity = nil

	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()

	startPos := Position{}
	if team == info.Yellow {
		startPos = Position{X: -200, Y: 0, Z: 0, Angle: 0}
	} else {
		startPos = Position{X: 200, Y: 0, Z: 0, Angle: 0}
	}

	r.ResetIfPreviousOwner(r, g)

	ge := gi.Status.GetGameEvent()
	r.isBallOwner = (g.GetBallOwner() == r.robotID)

	if r.isBallOwner {
		switch r.stateMachine {
		case KICKOFFER_GOTO_POS:
			switch r.state {
			case 0:
				activity = ai.NewMoveToPosition(team, r.robotID, startPos)
				if activity.Achieved(&gi) {
					r.NextState(r)
				}
			case 1:
				activity = ai.NewAlign(team, r.robotID, ballPos, startPos)
				achieved := activity.Achieved(&gi)
				if achieved && ge.RefCommand == info.NORMAL_START{
					if team == info.Yellow {
						r.ToStateMachine(KICKOFFER_PASS_BALL, r)
					} else {
						r.ToStateMachine(KICKOFFER_SHOOT_GOAL, r)
					}
				}
			}
		case KICKOFFER_PASS_BALL:
			switch r.state {
			case 0:
				receiverPos, _ := gi.State.GetTeam(team)[receiverID].GetPosition()
				activity = ai.NewAlign(team, r.robotID, receiverPos, ballPos)
				achieved := activity.Achieved(&gi)
				if achieved {
					activity = ai.NewKickBall(team, r.robotID, receiverPos, ballPos)
					r.NextState(r)
				}
			case 1:
				existingActivity := g.GetActivity(r.robotID)
				if existingActivity != nil && existingActivity.Achieved(&gi) {
					r.ResetState(r)
				}
			}
		case KICKOFFER_SHOOT_GOAL:
			switch r.state {
			case 0:
				opponentGoal := Position{X: -5050, Y: 0, Z: 0, Angle: 0}
				activity = ai.NewAlign(team, r.robotID, opponentGoal, ballPos)
				if activity.Achieved(&gi) {
					activity = ai.NewKickBall(team, r.robotID, opponentGoal, ballPos)
				}
			}
		}
	} else { // not ball owner
		switch r.stateMachine {
		case KICKER_STAY:
			switch r.state {
			case 0: // STATE_START
				robotPos, _ := gi.State.GetTeam(team)[r.robotID].GetPosition()
				activity = ai.NewMoveToPosition(team, r.robotID, robotPos)
				r.NextState(r)
			case 1: // STATE_EXECUTING
				existingActivity := g.GetActivity(r.robotID)
				if existingActivity != nil && existingActivity.Achieved(&gi) {
					r.ResetState(r)
				}
			}
		}
	}

	if activity != nil {
		g.AddActivity(activity)
	}

	return activity
}
