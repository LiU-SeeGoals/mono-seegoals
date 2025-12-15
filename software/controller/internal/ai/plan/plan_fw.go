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

func (m *plannerFw) run() {
	way_points := []info.Position{
		// Triangle
		{X: 0, Y: 0, Z: 0, Angle: 0},
		{X: -1000, Y: 0, Z: 0, Angle: 0},
		{X: -1000, Y: -1000, Z: 0, Angle: 0},
	}
	index := 0

	gameInfo := <-m.incomingGameInfo
	fmt.Println(gameInfo.Status)

	for {
		time.Sleep(100 * time.Millisecond)

		// if m.activities[3] == nil {
		//fmt.Println(fmt.Sprintf("done with (%d) action (%s)", robot, m.team))
		//fmt.Println("next action: ", way_points[index])
		m.AddActivity(ai.NewMoveToPosition(m.team, info.ID(3), way_points[index]))
		// index = (index + 1) % len(way_points)
		// }
	}
}
