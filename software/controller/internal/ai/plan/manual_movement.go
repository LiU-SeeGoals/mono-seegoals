package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const plannerManualMovementLoopPeriod = 100 * time.Millisecond

type plannerManualMovement struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

func NewPlannerManualMovement(team info.Team) *plannerManualMovement {
	return &plannerManualMovement{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *plannerManualMovement) Init(
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

func (m *plannerManualMovement) run() {

	gameInfo := <-m.incomingGameInfo
	fmt.Println(gameInfo.Status)

	for {
		command := client.GetCommand(helper.MOVE_ROBOT)
		// fmt.Println(len(commands))
		tickStart := time.Now()

		if command != nil{
			fmt.Println("changing command")
			pos := info.Position{X: float64(command.X), Y: float64(command.Y), Z:0,Angle: 0}

			m.ActivityHandler.AddActivity(ai.NewMoveToPosition(m.team, info.ID(command.Id), pos))
		}	

		helper.PaceLoop(tickStart, plannerManualMovementLoopPeriod, "planner_rw")
	}
}
