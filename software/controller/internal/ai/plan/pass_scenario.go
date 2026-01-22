package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/plt"
)

type GameScenario struct {
	plannerCore
	start             time.Time
	ballOwner         ID
	previousBallOwner ID
	ballOwnerTeam     Team
}

func NewGameScenario(team Team) *GameScenario {
	return &GameScenario{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *GameScenario) Init(
	incoming <-chan GameInfo,
	activities *[TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team Team,
) {
	m.incomingGameInfo = incoming
	m.activities = activities // store pointer directly
	m.activity_lock = lock
	m.team = team

	go m.run()
}

func (g *GameScenario) kicker(myID ID, receiver ID, gi GameInfo, statemachine int, state int) (int, int) {
	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()
	var activity ai.Activity
	receiverPos, _ := gi.State.GetTeam(g.team)[receiver].GetPosition()

	if g.previousBallOwner == myID {
		statemachine = 0
		state = 0
	}

	if g.ballOwner == myID {
		if statemachine == 0 { // fetch ball
			if state == 0 {
				activity = ai.NewAlign(g.team, myID, receiverPos, ballPos)
				state++
			} else if state == 1 {
				activity = g.GetActivity(myID)
				if activity.Achieved(&gi) {
					statemachine = 1
					state = 0
				}
			}
		} else if statemachine == 1 { // pass ball
			if state == 0 {
				activity = ai.NewKickBall(g.team, myID, ballPos)
				state++
			} else if state == 1 {
				activity = g.GetActivity(myID)
				if activity.Achieved(&gi) {
					state = 0
				}
			}
		}
	} else { // not ball owner
		if statemachine == 0 { // stay in place
			if state == 0 {
				robotPos, _ := gi.State.GetTeam(g.team)[myID].GetPosition()
				activity = ai.NewMoveToPosition(g.team, myID, robotPos)
				state++
			} else if state == 1 {
				activity = g.GetActivity(myID)
				fmt.Println(myID)
				if activity.Achieved(&gi) {
					state = 0
				}
			}
		}
	}

	g.AddActivity(activity)

	return statemachine, state
}

func (g *GameScenario) receiver(myID ID, gi GameInfo, statemachine int, state int) (int, int) {
	var activity ai.Activity = nil

	//ball := gi.State.GetTrackedBall()
	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()
	opponentGoal := Position{X: 1500, Y: 1000, Z: 0, Angle: 0}

	if g.previousBallOwner == myID {
		statemachine = 0
		state = 0
	}

	if g.ballOwner == myID {
		if statemachine == 0 { // turn to goal
			if state == 0 {
				activity = ai.NewAlign(g.team, myID, ballPos, opponentGoal)
				state++
			} else if state == 1 {
				activity = g.GetActivity(myID)
				if activity.Achieved(&gi) {
					state = 0
					statemachine = 1
				}
			}
		} else if statemachine == 1 { // kick ball into goal
			if state == 0 {
				activity = ai.NewKickBall(g.team, myID, ballPos)
				state++
			} else if state == 1 {
				activity = g.GetActivity(myID)
				if activity.Achieved(&gi) {
					state = 0
				}
			}
		}
	} else { // not ball owner
		if statemachine == 0 { // go to left field
			if state == 0 {
				wantedPos := Position{X: 1500, Y: 1000, Z: 0, Angle: 0}
				activity = ai.NewAlign(g.team, myID, wantedPos, ballPos)
				state++
			} else if state == 1 {
				activity = g.GetActivity(myID)
				if activity.Achieved(&gi) {
					state = 0
					statemachine = 1
				}
			}
		} else if statemachine == 1 { // watch ball and take ownership if close
			if state == 0 {
				wantedPos := Position{X: 1500, Y: 1000, Z: 0, Angle: 0}
				activity = ai.NewAlign(g.team, myID, wantedPos, ballPos)
				state++
			} else if state == 1 {
				myPos, _ := gi.State.GetTeam(g.team)[myID].GetPosition()
				if ballPos.Dist2d(myPos) < 800 {
					g.previousBallOwner = g.ballOwner
					g.ballOwner = myID
					statemachine = 0
					state = 0
				}
			}
		}
	}

	if activity != nil {
		g.AddActivity(activity)
	}

	return statemachine, state
}

func (g *GameScenario) run() {
	gi := <-g.incomingGameInfo
	plt.Init()

	kicker := info.ID(3)
	receiver := info.ID(1)
	active_robots := []int{int(kicker), int(receiver)}

	kicker_statemachine := 0
	kicker_state := 0

	receiver_statemachine := 0
	receiver_state := 0

	for {
		if g.HandleRef(&gi, active_robots) {
			continue
		}

		kicker_statemachine, kicker_state = g.kicker(kicker, receiver, gi, kicker_statemachine, kicker_state)
		receiver_statemachine, receiver_state = g.receiver(receiver, gi, receiver_statemachine, receiver_state)
	}

}
