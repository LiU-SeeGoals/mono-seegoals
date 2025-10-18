package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type TestGoalie struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

func NewTestGoalie(team info.Team) *TestGoalie {
	return &TestGoalie{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *TestGoalie) Init(
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

func (g *TestGoalie) run() {
	for {
		fmt.Println("TestGoalie running")
		// No need for slow brain to be fast
		time.Sleep(100 * time.Millisecond)

		// If there is no activity we add a goalie
		if g.activities[2] == nil {
			g.AddActivity(ai.NewGoalie(g.team, 2))
		}
	}

}
