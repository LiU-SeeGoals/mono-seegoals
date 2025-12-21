package ai

import (
	"sync"
	"time"
	"fmt"
	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/plt"
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

	state := 0
	num_states := 2
	gi := <-g.incomingGameInfo
	plt.Init()


	for{
		robot2, err := gi.State.GetTeam(g.team)[1].GetPosition()
		if err != nil{
			fmt.Println(err)
		}

		var activity ai.Activity

		if state == 0 {
			activity = ai.NewAlignBall(g.team, 3, robot2)

		}else if state == 1 {
			activity = ai.NewKickBall(g.team, 3)
		}

		g.AddActivity(activity)
		if activity.Achieved(&gi){
			state += (1 + state) % num_states
		}
	}

}
