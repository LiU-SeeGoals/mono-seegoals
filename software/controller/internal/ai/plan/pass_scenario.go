package ai

import (
	"fmt"
	"math"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

type GameScenario struct {
	plannerCore
	start             time.Time
	ballOwner         ID
	previousBallOwner ID
	ballOwnerTeam     Team
}

func NewGameScenario(team Team) *GameScenario {
	return &GameScenario{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *GameScenario) Init(
	incoming <-chan GameInfo,
	activities *[TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team Team,
) {
	m.incomingGameInfo = incoming
	m.ActivityHandler.Activities = activities
	m.ActivityHandler.Activity_lock = lock
	m.team = team

	go m.run()
}

func (g *GameScenario) getRobotClosestToBall(activeRobots []info.ID) info.ID {

	gi := <-g.incomingGameInfo
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	dist := math.Inf(1)

	clostestId := info.ID(0)
	for _, id := range activeRobots {
		robotPos, _ := gi.State.GetRobotPosition(g.team, id)
		ballrobotDist := ballPos.Dist2d(robotPos)
		if ballrobotDist < dist {
			clostestId = id
			dist = ballrobotDist
		}
	}
	return clostestId
}

func (m *GameScenario) run() {

	way_points := []info.Position{
		// Triangle
		{X: -1000, Y: 0, Z: 0, Angle: 0},
		{X: -2800, Y: 0, Z: 0, Angle: 0},
	}
	index := 0
	robots := []int{4}

	gameInfo := <-m.incomingGameInfo
	fmt.Println(gameInfo.Status)

	for {
		robotPos, _ := gameInfo.State.GetRobotPosition(m.team, 4)
		fmt.Println(robotPos.Dist2d(way_points[index]))
		fmt.Println(robotPos)
		fmt.Println(way_points[index])
		time.Sleep(50 * time.Millisecond)

		robot := robots[0]
		m.ActivityHandler.AddActivity(ai.NewMoveToPosition(m.team, info.ID(robot), way_points[index]))

		if robotPos.Dist2d(way_points[index]) < 50 {
			fmt.Println(fmt.Sprintf("done with (%d) action (%s)", robot, m.team))
			fmt.Println("next action: ", way_points[index])
			index = (index + 1) % len(way_points)
		}
	}
}
