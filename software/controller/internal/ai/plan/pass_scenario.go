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

var once bool
var activity ai.Activity

func (g *Pass) run() {

	state := 0
	num_states := 2
	gi := <-g.incomingGameInfo
	plt.Init()
	once = false
	prev_state := 0


	for{
		robot2, err := gi.State.GetTeam(g.team)[1].GetPosition()
		if err != nil{
			fmt.Println(err)
		}

		// fmt.Println(state)
		if state == 0 {
			activity = ai.NewAlignBall(g.team, 3, robot2)
			once = false

		} else if state == 1 {
			if (!once){
				ball := gi.State.GetBall()
				ballPos, _ := ball.GetEstimatedPosition()
				activity = ai.NewKickBall(g.team, 3, ballPos)
				fmt.Println("Once init")
			}
			once = true
		}

		g.AddActivity(activity)
		prev_state = state
		if activity.Achieved(&gi){
			state = (1 + state) % num_states
			if (state != prev_state){
				fmt.Printf("Switching state to %d\n", state)
			}
		}
	}

}
