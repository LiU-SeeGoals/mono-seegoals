package ai

import (
	"math"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type plannerContainer struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

func NewPlannerContainer(team info.Team) *plannerContainer {
	return &plannerContainer{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *plannerContainer) Init(
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
func (m *plannerContainer) run() {
	for {
		time.Sleep(100 * time.Millisecond)

		// if m.activities[0] == nil {
		//
		// 	activityQueue := []ai.Activity{
		// 		ai.NewMoveToPosition(m.team, 0, info.Position{X: 0, Y: 2000, Z: 0, Angle: 0}),
		// 		ai.NewMoveToPosition(m.team, 0, info.Position{X: 0, Y: 0, Z: 0, Angle: 0}),
		// 		ai.NewMoveToPosition(m.team, 0, info.Position{X: 2000, Y: 0, Z: 0, Angle: 0}),
		// 	}
		// 	queue := ai.NewActivityQueue(0, activityQueue)
		// 	m.AddActivity(queue)
		// }

		// if m.activities[1] == nil {
		//
		// 	activityLoop := []ai.Activity{
		// 		ai.NewMoveToPosition(m.team, 1, info.Position{X: 2000, Y: 2000, Z: 0, Angle: -math.Pi}),
		// 		ai.NewMoveToPosition(m.team, 1, info.Position{X: -2000, Y: 2000, Z: 0, Angle: math.Pi}),
		// 		ai.NewMoveToPosition(m.team, 1, info.Position{X: -2000, Y: -2000, Z: 0, Angle: -math.Pi/2}),
		// 		ai.NewMoveToPosition(m.team, 1, info.Position{X: 2000, Y: -2000, Z: 0, Angle: math.Pi/2}),
		// 		ai.NewMoveToPosition(m.team, 1, info.Position{X: 2000, Y: 2000, Z: 0, Angle: -math.Pi/4}),
		// 		ai.NewMoveToPosition(m.team, 1, info.Position{X: -2000, Y: -2000, Z: 0, Angle: math.Pi/4}),
		// 		ai.NewMoveToPosition(m.team, 1, info.Position{X: 2000, Y: -2000, Z: 0, Angle: -math.Pi/2}),
		// 		ai.NewMoveToPosition(m.team, 1, info.Position{X: -2000, Y: 2000, Z: 0, Angle: math.Pi/4}),
		// 	}
		// 	loop := ai.NewActivityLoop(1, activityLoop)
		// 	m.AddActivity(loop)
		// }

		if m.activities[2] == nil {

			activityLoop := []ai.Activity{
				ai.NewMoveToBall(m.team, 2),
				ai.NewMoveWithBallToPosition(m.team, 2, info.Position{X: -2000, Y: 0, Z: 0, Angle: math.Pi/4}),
				// ai.NewMoveWithBallToPosition(m.team, 2, info.Position{X: -2000, Y: 2000, Z: 0, Angle: math.Pi/4}),
				// ai.NewMoveWithBallToPosition(m.team, 2, info.Position{X: 2000, Y: 2000, Z: 0, Angle: math.Pi/4}),
			}
			loop := ai.NewActivityLoop(2, activityLoop)
			m.AddActivity(loop)
		}


	}

}
