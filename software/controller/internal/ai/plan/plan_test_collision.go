package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type plannerCollide struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

func NewPlannerCollide(team info.Team) *plannerCollide {
	return &plannerCollide{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *plannerCollide) Init(
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

// // This is the main loop of the AI in this slow brain
func (m *plannerCollide) run() {
	fmt.Println("slow brain started")

	way_points := []info.Position{
		{X: 0, Y: -1000, Z: 0, Angle: 0},
		{X: -1000, Y: 0, Z: 0, Angle: 0},
		{X: -1500, Y: -1500, Z: 0, Angle: 0},
		{X: -3000, Y: -1500, Z: 0, Angle: 0},
		{X: -3000, Y: 1500, Z: 0, Angle: 0},
	}

	for {
		// No need for slow brain to be fast
		time.Sleep(100 * time.Millisecond)

		//fmt.Println("slow, number of activities:", len(*m.activities))

		if m.activities[1] == nil {
			fmt.Println("done with action: ", m.team)
			// m.AddActivity(ai.NewMoveToPosition(m.team, 2, way_points[index]))
			m.AddActivity(ai.NewMoveToPosition(m.team, 1, way_points[1]))
		}

		if m.activities[2] == nil {
			fmt.Println("done with action: ", m.team)
			// m.AddActivity(ai.NewMoveToPosition(m.team, 2, way_points[index]))

			m.AddActivity(ai.NewMoveToPosition(m.team, 2, way_points[0]))
		}

		if m.activities[3] == nil {
			fmt.Println("done with action: ", m.team)
			// m.AddActivity(ai.NewMoveToPosition(m.team, 2, way_points[index]))
			m.AddActivity(ai.NewMoveToPosition(m.team, 3, way_points[2]))
		}

		if m.activities[4] == nil {
			fmt.Println("done with action: ", m.team)
			// m.AddActivity(ai.NewMoveToPosition(m.team, 2, way_points[index]))
			m.AddActivity(ai.NewMoveToPosition(m.team, 4, way_points[4]))
		}

		if m.activities[5] == nil {
			fmt.Println("done with action: ", m.team)
			// m.AddActivity(ai.NewMoveToPosition(m.team, 2, way_points[index]))
			m.AddActivity(ai.NewMoveToPosition(m.team, 5, way_points[3]))
		}

	}
}

// func (m *plannerCollide) run() {
// 	fmt.Println("slow brain started")

// 	// Seed the random number generator once
// 	rand.Seed(time.Now().UnixNano())
// 	a := 4000
// 	b := 2000

// 	for {
// 		// Slow down the loop so we don't hammer the CPU
// 		time.Sleep(100 * time.Millisecond)

// 		// For each robot (using indices as in the original code), if its activity is nil,
// 		// generate a random waypoint and assign a new move activity.
// 		if m.activities[2] == nil {
// 			wp := info.Position{
// 				X:     float64(rand.Intn(a) - 3000), // random between -3000 and 3000
// 				Y:     float64(rand.Intn(6001) - 3000),
// 				Z:     0,
// 				Angle: 0,
// 			}
// 			fmt.Println("done with action: ", m.team, "robot 2 moving to", wp)
// 			m.AddActivity(ai.NewMoveToPositionWithCollisionAvoidance(m.team, 2, wp, 2000))
// 		}
// 		if m.activities[1] == nil {
// 			wp := info.Position{
// 				X:     float64(rand.Intn(a) - b),
// 				Y:     float64(rand.Intn(a) - b),
// 				Z:     0,
// 				Angle: 0,
// 			}
// 			fmt.Println("done with action: ", m.team, "robot 1 moving to", wp)
// 			m.AddActivity(ai.NewMoveToPositionWithCollisionAvoidance(m.team, 1, wp, 2000))
// 		}
// 		if m.activities[4] == nil {
// 			wp := info.Position{
// 				X:     float64(rand.Intn(a) - b),
// 				Y:     float64(rand.Intn(a) - b),
// 				Z:     0,
// 				Angle: 0,
// 			}
// 			fmt.Println("done with action: ", m.team, "robot 4 moving to", wp)
// 			m.AddActivity(ai.NewMoveToPositionWithCollisionAvoidance(m.team, 4, wp, 2000))
// 		}
// 		if m.activities[5] == nil {
// 			wp := info.Position{
// 				X:     float64(rand.Intn(a) - b),
// 				Y:     float64(rand.Intn(a) - b),
// 				Z:     0,
// 				Angle: 0,
// 			}
// 			fmt.Println("done with action: ", m.team, "robot 5 moving to", wp)
// 			m.AddActivity(ai.NewMoveToPositionWithCollisionAvoidance(m.team, 5, wp, 2000))
// 		}
// 		if m.activities[3] == nil {
// 			wp := info.Position{
// 				X:     float64(rand.Intn(a) - b),
// 				Y:     float64(rand.Intn(a) - b),
// 				Z:     0,
// 				Angle: 0,
// 			}
// 			fmt.Println("done with action: ", m.team, "robot 3 moving to", wp)
// 			m.AddActivity(ai.NewMoveToPositionWithCollisionAvoidance(m.team, 3, wp, 2000))
// 		}
// 	}
// }

// package demos

// import (
// 	"time"

// 	"github.com/LiU-SeeGoals/controller/internal/ai"
// 	slow_brain "github.com/LiU-SeeGoals/controller/internal/ai/slow_brain"
// 	"github.com/LiU-SeeGoals/controller/internal/client"
// 	"github.com/LiU-SeeGoals/controller/internal/config"
// 	"github.com/LiU-SeeGoals/controller/internal/info"
// 	"github.com/LiU-SeeGoals/controller/internal/simulator"
// )

// func Scenario() {
// 	// This avoid the "No position in history" error for robots
// 	presentYellow := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
// 	presentBlue := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
// 	simController := simulator.NewSimControl()
// 	simController.SetPresentRobots(presentYellow, presentBlue)

// 	gameInfo := info.NewGameInfo(10)
// 	ssl_receiver := client.NewSSLClient(config.GetSSLClientAddress())

// 	// Yellow team
// 	slowBrainYellow := slow_brain.NewPlannerCollide(info.Yellow)
// 	fastBrainYellow := ai.NewFastBrainGO()

// 	aiYellow := ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)

// 	simClientYellow := client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)

// 	// Blue team
// 	// slowBrainBlue := slow_brain.NewPlanner1(info.Blue)
// 	// fastBrainBlue := ai.NewFastBrainGO()

// 	// aiBlue := ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

// 	// simClientBlue := client.NewSimClient(config.GetSimBlueTeamAddress(), gameInfo)

// 	// Some sim setup for debugging ai behaviour
// 	presentYellow = []int{1, 2, 3, 4, 5}
// 	presentBlue = []int{}
// 	simController.SetPresentRobots(presentYellow, presentBlue)
// 	simController.TeleportRobot(2000.0, 0, 1, info.Yellow)
// 	simController.TeleportRobot(0, 2000.0, 2, info.Yellow)
// 	simController.TeleportRobot(1500, 1500.0, 3, info.Yellow)
// 	simController.TeleportRobot(-3000, -1500.0, 4, info.Yellow)
// 	simController.TeleportRobot(-3000, 1500.0, 5, info.Yellow)
// 	simController.TeleportRobot(200.0, 0, 1, info.Blue)

// 	start_time := time.Now().UnixMilli()
// 	for {
// 		playTime := time.Now().UnixMilli() - start_time
// 		// fmt.Println("playTime: ", playTime)
// 		ssl_receiver.UpdateState(gameInfo, playTime)
// 		//fmt.Println(gameInfo)

// 		yellow_actions := aiYellow.GetActions(gameInfo)
// 		simClientYellow.SendActions(yellow_actions)

// 		//blue_actions := aiBlue.GetActions(gameInfo)
// 		//simClientBlue.SendActions(blue_actions)

// 		// terminal_messages := []string{"Scenario"}

// 		// if len(blue_actions) > 0 {
// 		// 	client.UpdateWebGUI(gameInfo, blue_actions, terminal_messages)
// 		// }
// 	}
// }
