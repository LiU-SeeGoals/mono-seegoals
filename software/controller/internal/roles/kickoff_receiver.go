package roles

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	KICKOFF_RECEIVER_GOTO_POS = 0
	KICKOFF_RECEIVER_GOTO_GOAL = 0
)

type KickoffReceiver struct {
	*RobotRole
}

func NewKickoffReceiver(robotID ID, gameScenario any) *KickoffReceiver {
	return &KickoffReceiver{
		RobotRole: NewRobotRole(robotID, "kickoff receiver", gameScenario),
	}
}

func (r *KickoffReceiver) GetStateMachineName() string {
	if r.isBallOwner {
		switch r.stateMachine {
		case KICKOFF_RECEIVER_GOTO_POS:
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

func (r *KickoffReceiver) StateMachine(gi GameInfo, team Team, g GameScenarioInterface, kickofferID ID) ai.Activity {
	var activity ai.Activity = nil

	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()
	startPos := Position{X: -400, Y: 1500, Z: 0, Angle: 0}

	r.ResetIfPreviousOwner(r, g)

	r.isBallOwner = (g.GetBallOwner() == r.robotID)

	if r.isBallOwner {
		switch r.stateMachine {
		case KICKOFF_RECEIVER_GOTO_GOAL:
			switch r.state {
			case 0:
				opponentGoal := Position{X: 5050, Y: 0, Z: 0, Angle: 0}
				activity = ai.NewAlign(team, r.robotID, opponentGoal, ballPos)
				if activity.Achieved(&gi) {
					activity = ai.NewKickBall(team, r.robotID, ballPos)
				}
			}
		}
	} else {
		switch r.stateMachine {
		case KICKOFF_RECEIVER_GOTO_POS:
			switch r.state {
			case 0:
				activity = ai.NewMoveToPosition(team, r.robotID, startPos)
				if activity.Achieved(&gi) {
					r.NextState(r)
				}
			case 1:
				kickofferPos, _ := gi.State.GetTeam(team)[kickofferID].GetPosition()
				activity = ai.NewAlign(team, r.robotID, kickofferPos, startPos)
				if activity.Achieved(&gi) {
					r.NextState(r)
				}
			case 2:
				kickofferPos, _ := gi.State.GetTeam(team)[kickofferID].GetPosition()
				activity = ai.NewAlign(team, r.robotID, kickofferPos, startPos)
				myPos, _ := gi.State.GetTeam(team)[r.robotID].GetPosition()
				if ballPos.Dist2d(myPos) < 500 {
					r.TakeOwnership(r, g, "kickoff receiver close to ball")
				}
			}
		}
	}

	if activity != nil {
		g.AddActivity(activity)
	}

	return activity
}
