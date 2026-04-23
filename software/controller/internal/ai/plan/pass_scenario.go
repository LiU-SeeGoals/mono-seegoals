package ai

import (
	"fmt"
	"math"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
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
	m.Active = true

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

func (g *GameScenario) run() {

	activeRobots := []info.ID{1, 3}
	kickers := make(map[info.ID]*roles.OffenseRole)

	gi := <-g.incomingGameInfo
	// refereeHandler := referee.NewRefereeHandler(&gi, activeRobots, g.team, &g.ActivityHandler)
	for _, id := range activeRobots {
		kicker := roles.NewOffenseRole(id, g.ActivityHandler, &gi, g.team)
		kicker.Init()
		kickers[id] = kicker
	}

	fmt.Println(gi.Status)

	for g.Active {
		tickStart := time.Now()
		// handleRef := refereeHandler.HandleReferee()
		// if handleRef {
		// 	helper.PaceLoop(tickStart, helper.PlannerLoopPeriod, "pass_scenario")
		// 	continue
		// }
		// Only coordinate robot roles, trigger ball events

		closestId := g.getRobotClosestToBall(activeRobots)

		for _, id := range activeRobots {
			if id != closestId {
				kickers[id].TriggerEvent("BALL_LOST")
			}
		}

		kickers[closestId].TriggerEvent("BALL_OWNER")

		for _, kicker := range kickers {
			kicker.Run()
		}

		// Defense strategy

		// Etc...
		helper.PaceLoop(tickStart, helper.PlannerLoopPeriod, "pass_scenario")
	}
}

func (m *GameScenario) Kill() {
	fmt.Println("Killing planner goalie")
	m.Active = false
}
