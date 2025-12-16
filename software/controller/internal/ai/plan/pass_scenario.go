package ai

import (
	"sync"
	"time"
	// "fmt"
	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

type Pass struct {
	plannerCore
	start    time.Time
}

func NewPass(team Team) *Pass {
	return &Pass{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *Pass) Init(
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

func (g *Pass) run() {

	// gameInfo <-incoming
	// gameInfo := <-g.incomingGameInfo


	for{
		if g.activities[3] == nil {
			queue := ai.NewActivityQueue(3, []ai.Activity{
				ai.NewAlignBall(g.team, 3),
				ai.NewKickBall(g.team, 3),
			})
			g.AddActivity(queue)
		}
		// g.AddActivity(ai.NewAlignBall(g.team, 3))
	}

}
