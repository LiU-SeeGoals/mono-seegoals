package roles

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
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

func (kr *KickerRole) StateMachine(gi GameInfo, team Team, g GameScenarioInterface, reciverId info.ID) ai.Activity {
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
					activity = ai.NewKickBall(team, kr.robotID, receiverPos, ballPos)
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
		robotPos, _ := gi.State.GetTeam(team)[kr.robotID].GetPosition()
		receiverPos, _ := gi.State.GetTeam(team)[reciverId].GetPosition()
		ab := receiverPos.Sub(&robotPos)
		tangent := Vec2{X: ab.X, Y: ab.Y}
		lookAtReciverAngle := tangent.Angle()

		switch kr.stateMachine {
		case KICKER_STAY:
			switch kr.state {
			case 0: // STATE_START
				followPos := Position{X: receiverPos.X, Y: -2000, Z: robotPos.Z, Angle: lookAtReciverAngle}
				activity = ai.NewMoveToPosition(team, kr.robotID, followPos)
				if ballPos.X < 0 || ballPos.Y < 0 {
					kr.TakeOwnership(kr, g, "ball on my side or close")
				}
			}
		}
	}

	if activity != nil {
		g.AddActivity(activity)
	}

	return activity
}
