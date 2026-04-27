package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

type plannerGoalie struct {
	plannerCore
	at_state          int
	start             time.Time
	max_time          time.Duration
	ballOwner         info.ID
	previousBallOwner info.ID
}

func NewPlannerGoalie(team info.Team) *plannerGoalie {
	return &plannerGoalie{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *plannerGoalie) Init(
	incoming <-chan info.GameInfo,
	activities *[info.TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	m.incomingGameInfo = incoming
	m.ActivityHandler.Activities = activities // store pointer directly
	m.ActivityHandler.Activity_lock = lock
	m.team = team
	m.start = time.Now()

	go m.run()
}

func (m *plannerGoalie) changeBallOwner(newOwner info.ID, reason string) {
	if m.ballOwner != newOwner {
		fmt.Printf("Ball Owner: %d -> %d (%s)\n", m.ballOwner, newOwner, reason)
		m.previousBallOwner = m.ballOwner
		m.ballOwner = newOwner
	}
}

func (m *plannerGoalie) AddActivity(activity ai.Activity) {
	m.ActivityHandler.AddActivity(activity)
}

func (m *plannerGoalie) GetActivity(id info.ID) ai.Activity {
	return m.ActivityHandler.GetActivity(id)
}

func (m *plannerGoalie) GetBallOwner() info.ID {
	return m.ballOwner
}

func (m *plannerGoalie) GetPreviousBallOwner() info.ID {
	return m.previousBallOwner
}

func (m *plannerGoalie) ChangeBallOwner(robotID info.ID, reason string) {
	m.changeBallOwner(robotID, reason)
}

func (m *plannerGoalie) GetTeam() info.Team {
	return m.team
}

// This is the main loop of the AI in this slow brain
func (m *plannerGoalie) run() {
	goalieID := info.ID(7)
	receiverID := info.ID(2)
	clearTarget := info.Position{X: 2000, Y: -1500, Z: 0, Angle: 0}
	goalieRole := roles.NewGoalieRole(goalieID, m.ActivityHandler, m.team, clearTarget)
	receiverRole := roles.NewReceiverRole(receiverID, m)
	goalieRole.Init()

	for {
		gi := <-m.incomingGameInfo
		goalieRole.SetGameInfo(gi)

		if goalieRole.HasBallControl(roles.GoalieBallControlRadius) {
			m.changeBallOwner(goalieID, "goalie has ball control")
			goalieRole.TriggerEvent("BALL_OWNER")
		} else {
			if m.ballOwner == goalieID {
				m.changeBallOwner(0, "goalie lost ball control")
			}
			goalieRole.TriggerEvent("BALL_LOST")
		}

		goalieRole.Run()
		receiverRole.ReceiverStateMachine(gi, m.team, m)

		if m.ballOwner == receiverID {
			ballPos, ballErr := gi.State.GetBall().GetEstimatedPosition()
			receiverPos, robotErr := gi.State.GetRobotPosition(m.team, receiverID)
			if ballErr == nil && robotErr == nil && ballPos.Dist2d(receiverPos) > 2000 {
				m.changeBallOwner(0, "receiver lost ball proximity")
			}
		}

		// No need for slow brain to be fast
		time.Sleep(5 * time.Millisecond)
	}
}
