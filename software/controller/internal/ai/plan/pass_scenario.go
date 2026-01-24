package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

const DEBUG = false

func debugf(format string, args ...interface{}) {
	if DEBUG {
		fmt.Printf(format, args...)
	}
}

type GameScenario struct {
	plannerCore
	start             time.Time
	ballOwner         ID
	previousBallOwner ID
	ballOwnerTeam     Team
}

func (g *GameScenario) changeBallOwner(newOwner ID, reason string) {
	if g.ballOwner != newOwner {
		fmt.Printf("Ball Owner: %d -> %d (%s)\n", g.ballOwner, newOwner, reason)
		g.previousBallOwner = g.ballOwner
		g.ballOwner = newOwner
	}
}

func (g *GameScenario) GetBallOwner() ID {
	return g.ballOwner
}

func (g *GameScenario) GetPreviousBallOwner() ID {
	return g.previousBallOwner
}

func (g *GameScenario) ChangeBallOwner(robotID ID, reason string) {
	g.changeBallOwner(robotID, reason)
}

func (g *GameScenario) GetTeam() Team {
	return g.team
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
	m.activities = activities
	m.activity_lock = lock
	m.team = team

	go m.run()
}

func (g *GameScenario) run() {
	kicker := info.ID(3)
	receiver := info.ID(1)
	//active_robots := []int{int(kicker), int(receiver)}

	kickerRole := roles.NewKickerRole(kicker, g)
	receiverRole := roles.NewReceiverRole(receiver, g)

	gi := <-g.incomingGameInfo
	// fmt.Println(gi.Status)

	for {
		gi = <-g.incomingGameInfo
		//if g.HandleRef(&gi, active_robots) {
		//	continue
		//}

		kickerRole.KickerStateMachine(gi, g.team, g)
		receiverRole.ReceiverStateMachine(gi, g.team, g)
	}
}
