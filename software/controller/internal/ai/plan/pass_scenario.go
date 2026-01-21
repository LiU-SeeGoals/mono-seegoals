package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/plt"
)

type Pass struct {
	plannerCore
	start time.Time
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

func (g *Pass) kickToHomie(robot ID, homie ID, gi GameInfo, state int, once bool, activity ai.Activity) (int, bool, ai.Activity, bool) {
	num_states := 2
	robot2, err := gi.State.GetTeam(g.team)[homie].GetPosition()
	done := false

	if err != nil {
		fmt.Println(err)
	}

	ball := gi.State.GetBall()
	ballPos, _ := ball.GetEstimatedPosition()

	if state == 0 {
		activity = ai.NewAlign(g.team, robot, robot2, ballPos)
		once = false

	} else if state == 1 {
		if !once {
			activity = ai.NewKickBall(g.team, robot, ballPos)
			fmt.Println("Once init")
		}
		once = true
	}

	if robot2.Dist2d(ballPos) < 800 {
		done = true
	}

	g.AddActivity(activity)

	if activity.Achieved(&gi) {
		state = (1 + state) % (num_states + 1)
	}

	if state == num_states {
		done = true
	}

	return state, once, activity, done
}

func (g *Pass) waitRecieve(robot ID, gi GameInfo, once bool, state int, activity ai.Activity) (int, bool, ai.Activity, bool) {
	num_states := 3
	robotPos, err := gi.State.GetTeam(g.team)[robot].GetPosition()
	done := false

	// State setup
	if err != nil {
		fmt.Println(err)
	}

	goal := Position{X: 0, Y: 0, Z: 0, Angle: 0}
	enemygoal := Position{X: 3050, Y: 0, Z: 0, Angle: 0}
	enemygoalSide := Position{X: 1500, Y: 1000, Z: 0, Angle: 0}

	ball := gi.State.GetBall()
	ballPos, err := ball.GetEstimatedPosition()

	// State handling

	if state == 0 {
		activity = ai.NewAlign(g.team, robot, goal, enemygoalSide)
	}
	if state == 1 {
		// if (!once){
		activity = ai.NewAlign(g.team, robot, enemygoal, ballPos)
		// }
		// once = true
	}
	if state == 2 {
		activity = ai.NewKickBall(g.team, robot, ballPos)
	}

	g.AddActivity(activity)

	// State transitions

	if ballPos.Dist2d(robotPos) > 800 {
		state = 0
		// once = false
	}
	if ballPos.Dist2d(robotPos) < 800 && activity.Achieved(&gi) && state == 0 {
		state = 1
	} else if state == 1 && activity.Achieved(&gi) {
		state = 2
		once = false
	} else if state == 2 && activity.Achieved(&gi) {
		state = 3
	} else if state == num_states {
		done = true
	}

	return state, once, activity, done
}

func (g *Pass) run() {

	stater1 := 0
	stater2 := 0
	gi := <-g.incomingGameInfo
	plt.Init()
	oncer1 := false
	oncer2 := false
	var actr1 ai.Activity
	var actr2 ai.Activity
	doner1 := false
	doner2 := false
	// vels := plotter.XYs{}

	fignum := 0
	active_robots := []int{1, 3}
	for {

		if g.HandleRef(&gi, active_robots) {
			continue
		}

		//ball := gi.State.GetBall()

		// ballVel, err := ball.GetVelocity2()
		ball := gi.State.GetTrackedBall()
		//ballPos, err := ball.GetTrackedPosition()
		ballVel, err := ball.GetTrackedVelocity()

		if err != true {
			ballVel = Position{X: 1000, Y: 1000, Z: 0, Angle: 0}
		} else if ballVel.Norm2d() > 0.001 && ballVel.Norm() < 1000 {
			// fmt.Println("ball vel", ballVel.Norm2d())
			// vels = append(vels, plotter.XY{X: float64(fignum), Y: ballVel.Norm()})
		}

		if !doner1 {
			stater1, oncer1, actr1, doner1 = g.kickToHomie(3, 1, gi, stater1, oncer1, actr1)
		}
		if !doner2 {
			stater2, oncer2, actr2, doner2 = g.waitRecieve(1, gi, oncer2, stater2, actr2)
		}
		if doner1 && stater2 == 0 && ballVel.Norm2d() < 0.01 {
			doner1 = false
			// doner2 = false
		}

		fignum += 1
		//fmt.Println(stater1, stater2)
		// if (fignum % 10 == 0){
		// 	plt.SaveFig(fmt.Sprintf("ballvel%d.png",fignum))
		// }
	}

}
