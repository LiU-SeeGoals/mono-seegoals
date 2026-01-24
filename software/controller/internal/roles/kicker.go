package roles

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	KICKER_FETCH_BALL = 0
	KICKER_PASS_BALL  = 1
	KICKER_STAY       = 0
)

type KickerRole struct {
	*RobotRole
}

func NewKickerRole(robotID ID, gameScenario interface{}) *KickerRole {
	return &KickerRole{
		RobotRole: NewRobotRole(robotID, "kicker", gameScenario),
	}
}

func (kr *KickerRole) GetStateMachineName() string {
	if kr.isBallOwner {
		switch kr.stateMachine {
		case KICKER_FETCH_BALL:
			return "FETCH_BALL"
		case KICKER_PASS_BALL:
			return "PASS_BALL"
		}
	} else {
		switch kr.stateMachine {
		case KICKER_STAY:
			return "KICKER_STAY"
		}
	}
	return fmt.Sprintf("SM_%d", kr.stateMachine)
}

func (kr *KickerRole) KickerStateMachine(gi GameInfo, team Team, g GameScenarioInterface) ai.Activity {
	var activity ai.Activity = nil

	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()
	receiverPos := Position{X: 1500, Y: 1500, Z: 0, Angle: 0}

	currentOwner := g.GetBallOwner()
	previousOwner := g.GetPreviousBallOwner()
	debugf("[DEBUG] Kicker %d: CurrentOwner: %d, PreviousOwner: %d, SM: %d, State: %d\n",
		kr.robotID, currentOwner, previousOwner, kr.stateMachine, kr.state)

	kr.ResetIfPreviousOwner(kr, g)

	kr.isBallOwner = (g.GetBallOwner() == kr.robotID)

	if kr.isBallOwner {
		switch kr.stateMachine {
		case KICKER_FETCH_BALL:
			switch kr.state {
			case 0: // STATE_START
				g.ChangeBallOwner(kr.robotID, "start of game")
				activity = ai.NewAlign(team, kr.robotID, receiverPos, ballPos)
				achieved := activity.Achieved(&gi)
				if achieved {
					kr.ToStateMachine(KICKER_PASS_BALL, kr)
				}
			}
		case KICKER_PASS_BALL:
			switch kr.state {
			case 0: // STATE_START
				activity = ai.NewAlign(team, kr.robotID, receiverPos, ballPos)
				achieved := activity.Achieved(&gi)
				if achieved {
					activity = ai.NewKickBall(team, kr.robotID, ballPos)
					kr.NextState(kr)
				}
			case 1: // STATE_EXECUTING
				existingActivity := g.GetActivity(kr.robotID)
				if existingActivity != nil && existingActivity.Achieved(&gi) {
					kr.ResetState(kr)
				}
			}
		}
	} else { // not ball owner
		switch kr.stateMachine {
		case KICKER_STAY:
			switch kr.state {
			case 0: // STATE_START
				robotPos, _ := gi.State.GetTeam(team)[kr.robotID].GetPosition()
				activity = ai.NewMoveToPosition(team, kr.robotID, robotPos)
				kr.NextState(kr)
			case 1: // STATE_EXECUTING
				if ballPos.X < 0 {
					kr.TakeOwnership(kr, g, "ball on my side")
				}
				existingActivity := g.GetActivity(kr.robotID)
				if existingActivity != nil && existingActivity.Achieved(&gi) {
					kr.ResetState(kr)
				}
			}
		}
	}

	if activity != nil {
		g.AddActivity(activity)
	}

	return activity
}
