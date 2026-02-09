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

type GameScenario2 struct {
	plannerCore
	start             time.Time
	ballOwner         ID
	previousBallOwner ID
	ballOwnerTeam     Team
}

func (g *GameScenario2) changeBallOwner(newOwner ID, reason string) {
	if g.ballOwner != newOwner {
		fmt.Printf("Ball Owner: %d -> %d (%s)\n", g.ballOwner, newOwner, reason)
		g.previousBallOwner = g.ballOwner
		g.ballOwner = newOwner
	}
}

func (g *GameScenario2) GetBallOwner() ID {
	return g.ballOwner
}

func (g *GameScenario2) GetPreviousBallOwner() ID {
	return g.previousBallOwner
}

func (g *GameScenario2) ChangeBallOwner(robotID ID, reason string) {
	g.changeBallOwner(robotID, reason)
}

func (g *GameScenario2) GetTeam() Team {
	return g.team
}

func NewGameScenario2(team Team) *GameScenario2 {
	return &GameScenario2{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *GameScenario2) Init(
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

func (g *GameScenario2) run() {
	kickerID := info.ID(7)

	kicker := roles.NewKickerRole(kickerID, g)
	kickoffer := roles.NewKickofferRole(kickerID, g)

	kickerDefenderPos := Position{X: -1000, Y: 0, Z: 0, Angle: 0}

	gi := <-g.incomingGameInfo
	fmt.Println(gi.Status)

	g.changeBallOwner(kickerID, "start of game")

	for {
		gi = <-g.incomingGameInfo

		ge := gi.Status.GetGameEvent()
		currentState := ge.GetCurrentState()
		previousState := ge.GetPreviousState()
		teamPossesor := ge.GetTeamWithPossession()

		switch currentState {
		case info.STATE_KICKOFF_PREPARATION:
			if previousState == info.STATE_HALTED || previousState == info.STATE_STOPPED {
				g.changeBallOwner(kickerID, "kickoff")
				kickoffer.ToStateMachine(0, kickoffer)
			}

			if teamPossesor == g.team {
				kickoffer.StateMachine(gi, g.team, g, 0)
			} else {
				g.AddActivity(ai.NewMoveToPosition(g.team, kickerID, kickerDefenderPos))
			}
		case info.STATE_FREE_KICK:
		case info.STATE_PENALTY_PREPARATION:
		case info.STATE_STOPPED:
			pos1 := Position{X: 2000, Y: 000, Z: 0, Angle: 0}
			g.AddActivity(ai.NewMoveToPosition(g.team, kickerID, pos1))
		case info.STATE_HALTED, info.STATE_TIMEOUT, info.STATE_BALL_PLACEMENT:
			// ball placement keep distance to ball only, halt should stop everything (and ball placement for now)
			str := fmt.Sprintf("game %s", currentState)
			g.changeBallOwner(0, str)
			kickerPos, _ := gi.State.GetTeam(g.team)[kickerID].GetPosition()
			g.AddActivity(ai.NewMoveToPosition(g.team, kickerID, kickerPos))
		case info.STATE_PLAYING:
			kicker.StateMachine(gi, g.team, g, 0)
		default:
		}
	}
}
