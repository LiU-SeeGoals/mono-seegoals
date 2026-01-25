package roles

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	FREEKICKER_GOTO_POS = 0
	FREEKICKER_PASS_BALL  = 1
	FREEKICKER_STAY       = 0
)

type FreekickerRole struct {
	*RobotRole
}

func NewFreekickerRole(robotID ID, gameScenario any) *FreekickerRole {
	return &FreekickerRole{
		RobotRole: NewRobotRole(robotID, "freekicker", gameScenario),
	}
}

func (r *FreekickerRole) GetStateMachineName() string {
	if r.isBallOwner {
		switch r.stateMachine {
		case FREEKICKER_GOTO_POS:
			return "GOTO_POS"
		case FREEKICKER_PASS_BALL:
			return "PASS_BALL"
		}
	} else {
		switch r.stateMachine {
		case FREEKICKER_STAY:
			return "STAY"
		}
	}
	return fmt.Sprintf("SM_%d", r.stateMachine)
}

func (r *FreekickerRole) StateMachine(gi GameInfo, team Team, g GameScenarioInterface, receiverID ID) ai.Activity {
	var activity ai.Activity = nil

	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()

	r.ResetIfPreviousOwner(r, g)

	r.isBallOwner = (g.GetBallOwner() == r.robotID)

	if r.isBallOwner {
		switch r.stateMachine {
		case FREEKICKER_GOTO_POS:
			switch r.state {
			case 0:
				ballPos.X -= -300
				ballPos.Y -= -300
				activity = ai.NewMoveToPosition(team, r.robotID, ballPos)
				if activity.Achieved(&gi) {
					r.NextState(r)
				}
			case 1:
				receiverPos, _ := gi.State.GetTeam(team)[receiverID].GetPosition()
				activity = ai.NewAlign(team, r.robotID, receiverPos, ballPos)
				achieved := activity.Achieved(&gi)
				if achieved {
					activity = ai.NewKickBall(team, r.robotID, ballPos)
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
