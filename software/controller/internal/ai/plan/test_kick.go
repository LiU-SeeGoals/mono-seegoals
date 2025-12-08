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
	m.activities = activities // store pointer directly
	m.activity_lock = lock
	m.team = team
	m.start = time.Now()

	go m.run()
}

func wait(ms time.Duration) {
	time.Sleep(ms)
}

func (g *TestKick) run() {
	for {
		wait(100 * time.Millisecond)
		if g.activities[0] == nil {
			queue := ai.NewActivityQueue(0, []ai.Activity{
				ai.NewMoveToBall(g.team, 0),
				ai.NewKickAtPosition(g.team, 0, info.Position{X: 2900, Y: 100}),
			})
			g.AddActivity(queue)
		}
	}

}
