package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

// ========================================================
// planner2 is a demo slow brain with referee handling
// ========================================================

type planner2 struct {
	plannerCore

	at_state int
	start    time.Time
	max_time time.Duration
	team     info.Team
	prev_ref info.RefCommand
}

func NewPlanner2(team info.Team) *planner2 {
	return &planner2{
		plannerCore: plannerCore{
			team: team,
		},
		team: team,
	}
}

func (m *planner2) Init(
	incoming <-chan info.GameInfo,
	activities *[info.TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	m.incomingGameInfo = incoming
	m.activities = activities // store pointer directly
	m.activity_lock = lock
	m.start = time.Now()

	go m.run()
}

// This is the main loop of the AI in this slow brain
func (m *planner2) run() {
	way_points := []info.Position{
		{X: 0, Y: 0, Z: 0, Angle: 0},
		{X: 0, Y: 1000, Z: 0, Angle: 0},
		{X: 1000, Y: 0, Z: 0, Angle: 0},
	}
	index := 0
	fmt.Println("slow brain 2: starting")

	active_id := []int{0, 1}

	for {
		// No need for slow brain to be fast
		time.Sleep(100 * time.Millisecond)

		gameInfo := <-m.incomingGameInfo
		// fmt.Println("slow brain 2: ")

		fmt.Println(gameInfo.Status.GetGameEvent())

		if m.HandleRef(&gameInfo, active_id) {
			continue
		}

		fmt.Println(m.activities[0])
		if m.activities[0] == nil {
			fmt.Println("done with action: ", m.team)
			m.AddActivity(ai.NewMoveToPosition(m.team, 0, way_points[index]))
			index = (index + 1) % len(way_points)
		}

	}
}
