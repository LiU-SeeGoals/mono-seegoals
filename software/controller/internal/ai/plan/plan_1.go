package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type planner1 struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

func NewPlanner1(team info.Team) *planner1 {
	return &planner1{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *planner1) Init(
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
func (m *planner1) run() {

	way_points := []info.Position{
		{X: 0, Y: 0, Z: 0, Angle: 0},
		{X: 0, Y: 1000, Z: 0, Angle: 0},
		{X: 1000, Y: 0, Z: 0, Angle: 0},
	}
	index := 0

	for {
		// No need for slow brain to be fast
		time.Sleep(100 * time.Millisecond)

		//fmt.Println("slow, number of activities:", len(*m.activities))

		if m.activities[2] == nil {
			fmt.Println("done with action: ", m.team)
			// time.Sleep(10000 * time.Millisecond)
			m.AddActivity(ai.NewMoveToPosition(m.team, 2, way_points[index]))
			index = (index + 1) % len(way_points)
		}
	}
}
