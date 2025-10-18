package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type plannerRw struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

func NewPlannerRw(team info.Team) *plannerRw {
	return &plannerRw{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *plannerRw) Init(
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

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 *                                                                                       *
 *                                                                                       *
 * This is Rasmus Wallin's file, touch it without asking and you shall meet your demise! *
 *                                                                                       *
 *                                                                                       *
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (m *plannerRw) run() {
	way_points := []info.Position{
		// Go between line
		//{X: -3575, Y: -4128, Z: 0, Angle: 0},
		//{X: -5558, Y: -4096, Z: 0, Angle: 0},
		// Go to pos
		//{X: -4195, Y: -3544, Z: 0, Angle: 0},
		// Triangle
		//{X: -5500, Y: -4100, Z: 0, Angle: 0},
		//{X: -5600, Y: -2600, Z: 0, Angle: 0},
		//{X: -4200, Y: -3400, Z: 0, Angle: 0},
		// Triangle 2
		{X: -2000, Y: 0, Z: 0, Angle: 0},
		{X: 500, Y: 2000, Z: 0, Angle: 150},
		{X: 2000, Y: 1000, Z: 0, Angle: 100},
	}
	index := 0
	succesfull_commands := 0
	robots := []int{0, 1, 3}

	gameInfo := <-m.incomingGameInfo
	fmt.Println(gameInfo.Status)

	for {
		// No need for slow brain to be fast
		time.Sleep(100 * time.Millisecond)

		//if m.HandleRef(&gameInfo, robots) {
		//	continue
		//}

        robot := robots[0]
        if m.activities[robot] == nil {
            fmt.Println(fmt.Sprintf("done with (%d) action (%s)", robot, m.team))
            fmt.Println("next action: ", way_points[index])
            fmt.Println("sent commands: ", succesfull_commands)
            m.AddActivity(ai.NewMoveToPosition(m.team, info.ID(robot), way_points[index]))
            index = (index + 1) % len(way_points)
            succesfull_commands++
        }

        robot = robots[1]
        if m.activities[robot] == nil {
            fmt.Println(fmt.Sprintf("done with (%d) action (%s)", robot, m.team))
            fmt.Println("next action: ", way_points[index])
            fmt.Println("sent commands: ", succesfull_commands)
            m.AddActivity(ai.NewMoveToPosition(m.team, info.ID(robot), way_points[index]))
            index = (index + 1) % len(way_points)
            succesfull_commands++
        }

        robot = robots[2]
        if m.activities[robot] == nil {
            fmt.Println(fmt.Sprintf("done with (%d) action (%s)", robot, m.team))
            fmt.Println("next action: ", way_points[index])
            fmt.Println("sent commands: ", succesfull_commands)
            m.AddActivity(ai.NewMoveToPosition(m.team, info.ID(robot), way_points[index]))
            index = (index + 1) % len(way_points)
            succesfull_commands++
        }
	}
}
