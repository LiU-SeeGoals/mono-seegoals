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
		// Triangle
		{X: 1000, Y: 1000, Z: 0, Angle: 0},
		{X: -1000, Y: 1000, Z: 0, Angle: 0},
		{X: 1000, Y: -1000, Z: 0, Angle: 0},
	}
	index := 0
	robots := []int{0}

	gameInfo := <-m.incomingGameInfo
	fmt.Println(gameInfo.Status)

	for {
		time.Sleep(100 * time.Millisecond)

		robot := robots[0]
		if m.activities[robot] == nil {
			fmt.Println(fmt.Sprintf("done with (%d) action (%s)", robot, m.team))
			fmt.Println("next action: ", way_points[index])
			m.AddActivity(ai.NewMoveToPosition(m.team, info.ID(robot), way_points[index]))
			index = (index + 1) % len(way_points)
		}
	}
}
