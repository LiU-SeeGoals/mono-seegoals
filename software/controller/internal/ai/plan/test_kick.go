package ai

import (
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type TestKick struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

func NewTestKick(team info.Team) *TestKick {
	return &TestKick{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *TestKick) Init(
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

func (g *TestKick) run() {
	for {
		<-g.incomingGameInfo
		if g.ActivityHandler.Activities[3] == nil {
			queue := ai.NewActivityQueue(3, []ai.Activity{
				ai.NewMoveToBall(g.team, 3),
				ai.NewKickAtPosition(g.team, 3, info.Position{X: 2900, Y: 100}),
			})
			g.ActivityHandler.AddActivity(queue)
		}
	}

}
