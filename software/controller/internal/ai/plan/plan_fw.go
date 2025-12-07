package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type plannerFw struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

func NewPlannerFw(team info.Team) *plannerFw {
	return &plannerFw{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *plannerFw) Init(
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

// This is the main loop of the AI in this slow brain
func (m *plannerFw) run() {
	// way_points := []info.Position{
	// 	// Go between line
	// 	//{X: -3575, Y: -4128, Z: 0, Angle: 0},
	// 	//{X: -5558, Y: -4096, Z: 0, Angle: 0},
	// 	// Go to pos
	// 	//{X: -4195, Y: -3544, Z: 0, Angle: 0},
	// 	// Triangle
	// 	//{X: -5500, Y: -4100, Z: 0, Angle: 0},
	// 	//{X: -5600, Y: -2600, Z: 0, Angle: 0},
	// 	//{X: -4200, Y: -3400, Z: 0, Angle: 0},
	// 	// Triangle 2
	// 	{X: -2920, Y: -4100, Z: 0, Angle: 0},
	// 	{X: -5900, Y: -1950, Z: 0, Angle: 0},
	// 	{X: -4250, Y: -1950, Z: 0, Angle: 0},
	// }
	// index := 0
	// succesfull_commands := 0
	// //robots := []int{0}
	// robot := 1

	gameInfo := <-m.incomingGameInfo
	fmt.Println(gameInfo.Status)
	robot1_index := 0
	// robot0_index := 0

	way_points := []info.Position{
		{X: 4600, Y: 0, Z: 0, Angle: 0},
		{X: -4600, Y: 0, Z: 0, Angle: 0},
		{X: 2000, Y: -1000, Z: 0, Angle: 0},
		{X: -2000, Y: -1000, Z: 0, Angle: 0},
	}
	active_id := []int{0, 1, 3}
	enemy_goal := 0

	for {
		fmt.Println(gameInfo.Status)
		// No need for slow brain to be fast
		time.Sleep(100 * time.Millisecond)

		if m.HandleRef(&gameInfo, active_id) {
			continue
		}

		if gameInfo.Status.GetBlueTeamOnPositiveHalf() && m.team == info.Blue {
			enemy_goal = 0
		} else {
			enemy_goal = 1

		}

		if m.activities[3] == nil {
			fmt.Println("done with action for robot 1: ", m.team)
			m.AddActivity(ai.NewMoveToPosition(m.team, 3, way_points[2+enemy_goal]))
			robot1_index = (robot1_index + 1) % len(way_points)

		}

		if m.activities[0] == nil {
			fmt.Println("done with action for robot 1: ", m.team)
			m.AddActivity(ai.NewGoalie(m.team, 0))

		}

		// if m.activities[1] == nil {
		// 	fmt.Println("done with action for robot 0: ", m.team)
		// 	m.AddActivity(ai.NewGoalie(m.team, 1))
		// }

		if m.activities[1] == nil {
			fmt.Println("done with action: ", m.team)

			// If we have the ball, then dribble to the enemy goal
			possessor := gameInfo.State.GetBall().GetPossessor()

			if possessor != nil && possessor.GetID() == 1 {
				m.AddActivity(ai.NewMoveWithBallToPosition(m.team, 1, way_points[enemy_goal]))

			} else {
				m.AddActivity(ai.NewMoveToBall(m.team, 1))
			}

		}

		// if m.activities[1] == nil {
		// 	fmt.Println("done with action for robot 0: ", m.team)
		// 	m.AddActivity(ai.NewMoveToPosition(m.team, 1, way_points[robot0_index]))
		// 	robot0_index = (robot0_index + 1) % len(way_points)

		// }

		//for _, robot := range robots {
		// if m.activities[robot] == nil {
		// 	fmt.Println(fmt.Sprintf("done with (%d) action (%s)", robot, m.team))
		// 	fmt.Println("next action: ", way_points[index])
		// 	fmt.Println("sent commands: ", succesfull_commands)
		// 	m.AddActivity(ai.NewMoveToPosition(m.team, info.ID(robot), way_points[index]))
		// 	index = (index + 1) % len(way_points)
		// 	succesfull_commands++
		// }
		//}
	}
}
