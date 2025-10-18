package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type plannerTest struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

func NewPlannerTest(team info.Team) *plannerTest {
	return &plannerTest{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *plannerTest) Init(
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
func (m *plannerTest) run() {

	var id info.ID	
	activityList := []ai.Activity{
		ai.NewMoveToPosition(m.team, id, info.Position{X: 2000, Y: 0, Z: 0, Angle: 0}),
		ai.NewMoveToBall(m.team, id),
		ai.NewMoveWithBallToPosition(m.team, id, info.Position{X: 2000, Y: 0, Z: 0, Angle: 0}),
		ai.NewKickToPlayer(m.team, id, 1),

	}
	index := 0


	for {
		// No need for slow brain to be fast
		time.Sleep(100 * time.Millisecond)


		if m.activities[0] == nil {
			fmt.Println("Adding activity: ", activityList[index].String())
			m.AddActivity(activityList[index])
			index = (index + 1) % len(activityList)
		}
	}

}


// // a slice of all the activities, movetoposition, movetoball etc
// 	activityList := []ai.Activity{
// 		ai.NewMoveToPosition(m.team, 0, info.Position{X: 0, Y: 0, Z: 0, Angle: 0}),
// 		ai.NewMoveToBall(m.team, 0),
// 		ai.NewMoveWithBallToPosition(m.team, 0, info.Position{X: 0, Y: 0, Z: 0, Angle: 0}),
// 		ai.NewKickToPlayer(m.team, 0, 1),
// 	}
// 	index := 0
//
//
// 	for {
// 		// No need for slow brain to be fast
// 		time.Sleep(100 * time.Millisecond)
//
//
// 		if len(*m.activities) == 0 {
// 			fmt.Println("Adding activity: ", activityList[index].String())
// 			m.AddActivity(activityList[index])
// 			index = (index + 1) % len(activityList)
// 		}
// 	}
