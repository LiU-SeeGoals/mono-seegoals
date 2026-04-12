package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type plannerGoalie struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

func NewPlannerGoalie(team info.Team) *plannerGoalie {
	return &plannerGoalie{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *plannerGoalie) Init(
	incoming <-chan info.GameInfo,
	activities *[info.TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	m.incomingGameInfo = incoming
	m.ActivityHandler.Activities = activities // store pointer directly
	m.ActivityHandler.Activity_lock = lock
	m.team = team
	m.start = time.Now()

	go m.run()
}

// This is the main loop of the AI in this slow brain
func (m *plannerGoalie) run() {

	for {
		tickStart := time.Now()

		//fmt.Println("slow, number of activities:", len(*m.activities))

		if m.ActivityHandler.Activities[7] == nil {
			fmt.Println("done with action: ", m.team)
			m.ActivityHandler.AddActivity(ai.NewGoalie(m.team, 7))
		}

		helper.PaceLoop(tickStart, helper.PlannerLoopPeriod, "planner_goalie")
	}
}
