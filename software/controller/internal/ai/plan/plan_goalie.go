package ai

import (
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

type plannerGoalie struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
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

// This is the main loop of the AI in this slow brain
func (m *plannerGoalie) run() {
	goalieID := info.ID(7)
	clearTarget := info.Position{X: 0, Y: 1000, Z: 0, Angle: 0}
	goalieRole := roles.NewGoalieRole(goalieID, m.ActivityHandler, m.team, clearTarget)
	goalieRole.Init()

	for {
		gi := <-m.incomingGameInfo
		goalieRole.SetGameInfo(gi)

		if goalieRole.HasBallControl(roles.GoalieBallControlRadius) {
			goalieRole.TriggerEvent("BALL_OWNER")
		} else {
			goalieRole.TriggerEvent("BALL_LOST")
		}

		goalieRole.Run()

		// No need for slow brain to be fast
		time.Sleep(5 * time.Millisecond)
	}
}
