package ai

import (
	"fmt"
	"math"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/referee"
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
		// fmt.Println(id ,ballrobotDist)
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
	refereeHandler := referee.NewRefereeHandler(&gi, activeRobots, g.team, &g.ActivityHandler)
	for _, id := range activeRobots {
		kicker := roles.NewKickerRole2(id, g.ActivityHandler, &gi, g.team)
		kicker.Init()
		kickers[id] = kicker
	}

	fmt.Println(gi.Status)

	for {
		// ge := gi.Status.GetGameEvent()
		// fmt.Println(ge.CurrentState)
		// refereeCommand := gi.Status.GetGameEvent().RefCommand.String()
		// fmt.Println(refereeCommand)
		time.Sleep(time.Millisecond * 1)

		handleRef := refereeHandler.HandleReferee()
		// fmt.Println(handleRef)
		if handleRef{
			continue
		}
		// trackedBall := gi.State.GetTrackedBall();
		// ball := gi.State.GetBall();
		// ballPos, _ := ball.GetPosition()
		// fmt.Println("Untracked - Tracked: ", ballPos, "\t", trackedBall.Vel);

		// Only coordinate robot roles, trigger ball events

		// time.Sleep(time.Millisecond * 1)
		// Attack strategy

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
	}
}
