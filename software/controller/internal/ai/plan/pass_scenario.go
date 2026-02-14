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

func (g *GameScenario) run() {
	kickerID := info.ID(3)

	gi := <-g.incomingGameInfo
	kicker := roles.NewKickerRole2(kickerID, g.ActivityHandler, &gi, g.team)
	kicker.Init()
	// receiverID := info.ID(1)

	// kicker := roles.NewKickerRole(kickerID, g)
	// receiverRole := roles.NewReceiverRole(receiverID, g)
	// kickoffer := roles.NewKickofferRole(kickerID, g)
	// kickoffReceiver := roles.NewKickoffReceiver(receiverID, g)
	// freeKicker := roles.NewFreekickerRole(kickerID, g)
	//
	// kickerDefenderPos := Position{X: -1000, Y: 0, Z: 0, Angle: 0}
	// receiverDefenderPos := Position{X: -1000, Y: 1500, Z: 0, Angle: 0}

	fmt.Println(gi.Status)

	// g.changeBallOwner(kickerID, "start of game")

	for {
		time.Sleep(10);
		pos,_ := gi.State.GetRobot(kickerID, g.team).GetPosition()
		ballPos,_ := gi.State.GetBall().GetEstimatedPosition()
		if pos.Dist2d(ballPos) < 800{
			kicker.TriggerEvent("BALL_OWNER")
		}

		kicker.Run()
	}
}
