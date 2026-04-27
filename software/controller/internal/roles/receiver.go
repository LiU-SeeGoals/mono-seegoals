package roles

import (
	"fmt"
	"math"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	RECEIVER_TURN_TO_GOAL = 0
	RECEIVER_KICK_GOAL    = 1
	RECEIVER_POSITION     = 0
	RECEIVER_WATCH_BALL   = 1
)

type ReceiverRole struct {
	*RobotRole
}

func receiverHomePosition(team Team) Position {
	pos := Position{X: 1000, Y: 1500, Z: 0, Angle: 0}
	if team == Yellow {
		return pos.Rotate(math.Pi)
	}

	return pos
}

func NewReceiverRole(robotID ID, gameScenario interface{}) *ReceiverRole {
	return &ReceiverRole{
		RobotRole: NewRobotRole(robotID, "receiver", gameScenario),
	}
}

func (rr *ReceiverRole) GetStateMachineName() string {
	if rr.isBallOwner {
		switch rr.stateMachine {
		case RECEIVER_TURN_TO_GOAL:
			return "TURN_TO_GOAL"
		case RECEIVER_KICK_GOAL:
			return "KICK_GOAL"
		}
	} else {
		switch rr.stateMachine {
		case RECEIVER_POSITION:
			return "POSITION"
		case RECEIVER_WATCH_BALL:
			return "WATCH_BALL"
		}
	}
	return fmt.Sprintf("SM_%d", rr.stateMachine)
}

func (rr *ReceiverRole) ReceiverStateMachine(gi GameInfo, team Team, g GameScenarioInterface) ai.Activity {
	var activity ai.Activity = nil

	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()
	opponentGoal := gi.EnemyGoalCenter(team)

	currentOwner := g.GetBallOwner()
	previousOwner := g.GetPreviousBallOwner()
	debugf("[DEBUG] Receiver %d: CurrentOwner: %d, PreviousOwner: %d, SM: %d, State: %d\n",
		rr.robotID, currentOwner, previousOwner, rr.stateMachine, rr.state)

	rr.ResetIfPreviousOwner(rr, g)

	rr.isBallOwner = (g.GetBallOwner() == rr.robotID)

	if rr.isBallOwner {
		switch rr.stateMachine {
		case RECEIVER_TURN_TO_GOAL:
			switch rr.state {
			case 0: // STATE_START
				activity = ai.NewAlign(team, rr.robotID, opponentGoal, ballPos)
				if activity.Achieved(&gi) {
					rr.ToStateMachine(RECEIVER_KICK_GOAL, rr)
				}
			}
		case RECEIVER_KICK_GOAL:
			switch rr.state {
			case 0: // STATE_START
				activity = ai.NewKickBall(team, rr.robotID, opponentGoal, ballPos)
				rr.NextState(rr)
			case 1: // STATE_EXECUTING
				existingActivity := g.GetActivity(rr.robotID)
				if existingActivity != nil && existingActivity.Achieved(&gi) {
					rr.ToStateMachine(RECEIVER_TURN_TO_GOAL, rr)
				}
			}
		}
	} else { // not ball owner
		switch rr.stateMachine {
		case RECEIVER_POSITION:
			switch rr.state {
			case 0: // STATE_START
				wantedPos := receiverHomePosition(team)
				activity = ai.NewMoveToPosition(team, rr.robotID, wantedPos)
				rr.NextState(rr)
			case 1: // STATE_EXECUTING
				existingActivity := g.GetActivity(rr.robotID)
				if existingActivity != nil {
					achieved := existingActivity.Achieved(&gi)
					if achieved {
						rr.ToStateMachine(RECEIVER_WATCH_BALL, rr)
					}
				}
			}

		case RECEIVER_WATCH_BALL:
			switch rr.state {
			case 0: // STATE_START
				myPos, _ := gi.State.GetTeam(team)[rr.robotID].GetPosition()
				activity = ai.NewAlign(team, rr.robotID, ballPos, myPos)
				rr.NextState(rr)
			case 1: // STATE_EXECUTING
				myPos, _ := gi.State.GetTeam(team)[rr.robotID].GetPosition()
				if ballPos.Dist2d(myPos) < 2000 {
					rr.TakeOwnership(rr, g, "receiver close to ball")
				}
			}
		}
	}

	if activity != nil {
		g.AddActivity(activity)
	}

	return activity
}
