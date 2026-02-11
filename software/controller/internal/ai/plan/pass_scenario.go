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
	m.ActivityHandler.activities = activities
	m.ActivityHandler.activity_lock = lock
	m.team = team

	go m.run()
}

var refStateNames = map[info.RefState]string{
	info.STATE_HALTED:              "halted",
	info.STATE_STOPPED:             "stopped",
	info.STATE_PLAYING:             "playing",
	info.STATE_KICKOFF_PREPARATION: "kickoff_preparation",
	info.STATE_PENALTY_PREPARATION: "penalty_preparation",
	info.STATE_FREE_KICK:           "free_kick",
	info.STATE_TIMEOUT:             "timeout",
	info.STATE_BALL_PLACEMENT:      "ball_placement",
}

var oldRefState info.RefState = -1

func (g *GameScenario) run() {
	kickerID := info.ID(3)
	receiverID := info.ID(1)

	kicker := roles.NewKickerRole(kickerID, g)
	receiverRole := roles.NewReceiverRole(receiverID, g)
	kickoffer := roles.NewKickofferRole(kickerID, g)
	kickoffReceiver := roles.NewKickoffReceiver(receiverID, g)
	freeKicker := roles.NewFreekickerRole(kickerID, g)

	kickerDefenderPos := Position{X: -1000, Y: 0, Z: 0, Angle: 0}
	receiverDefenderPos := Position{X: -1000, Y: 1500, Z: 0, Angle: 0}

	gi := <-g.incomingGameInfo
	fmt.Println(gi.Status)

	g.changeBallOwner(kickerID, "start of game")

	for {
		gi = <-g.incomingGameInfo

		ge := gi.Status.GetGameEvent()
		currentState := ge.GetCurrentState()
		previousState := ge.GetPreviousState()
		teamPossesor := ge.GetTeamWithPossession()
		if currentState != oldRefState {
			fmt.Printf("\nReferee: %s -> %s (%s)\n", refStateNames[previousState], refStateNames[currentState], teamPossesor)
			oldRefState = currentState
		}

		switch currentState {
		case info.STATE_KICKOFF_PREPARATION:
			if previousState == info.STATE_HALTED || previousState == info.STATE_STOPPED {
				g.changeBallOwner(kickerID, "kickoff")
				kickoffer.ToStateMachine(0, kickoffer)
				kickoffReceiver.ToStateMachine(0, kickoffReceiver)
			}

			if teamPossesor == g.team {
				kickoffer.StateMachine(gi, g.team, g, receiverID)
				kickoffReceiver.StateMachine(gi, g.team, g, kickerID)
			} else {
				g.AddActivity(ai.NewMoveToPosition(g.team, kickerID, kickerDefenderPos))
				g.AddActivity(ai.NewMoveToPosition(g.team, receiverID, receiverDefenderPos))
			}
		case info.STATE_FREE_KICK:
			ballPos, _ := gi.State.GetBall().GetPosition()
			if teamPossesor == g.team {
				g.changeBallOwner(kickerID, "freekick our team")
				freeKicker.StateMachine(gi, g.team, g, kickerID)
				ballPos.X -= 300
				ballPos.Y -= 300
				g.AddActivity(ai.NewMoveToPosition(g.team, receiverID, ballPos))
			} else {
				g.changeBallOwner(0, "freekick other team")
				ballPos.X -= 600
				ballPos.Y -= 600
				g.AddActivity(ai.NewMoveToPosition(g.team, kickerID, ballPos))
				g.AddActivity(ai.NewMoveToPosition(g.team, receiverID, ballPos))
			}
		case info.STATE_PENALTY_PREPARATION:
		case info.STATE_STOPPED:
			pos1 := Position{X: -1000, Y: -500, Z: 0, Angle: 0}
			pos2 := Position{X: -2000, Y: 1000, Z: 0, Angle: 0}
			g.AddActivity(ai.NewMoveToPosition(g.team, kickerID, pos1))
			g.AddActivity(ai.NewMoveToPosition(g.team, receiverID, pos2))
		case info.STATE_HALTED, info.STATE_TIMEOUT, info.STATE_BALL_PLACEMENT:
			// ball placement should keep distance to ball only, halt should stop everything (and ball placement for now)
			str := fmt.Sprintf("game %s", currentState)
			g.changeBallOwner(0, str)
			kickerPos, _ := gi.State.GetTeam(g.team)[kickerID].GetPosition()
			receiverPos, _ := gi.State.GetTeam(g.team)[receiverID].GetPosition()
			g.AddActivity(ai.NewMoveToPosition(g.team, kickerID, kickerPos))
			g.AddActivity(ai.NewMoveToPosition(g.team, receiverID, receiverPos))
		case info.STATE_PLAYING:
			kicker.StateMachine(gi, g.team, g, receiverID)
			receiverRole.ReceiverStateMachine(gi, g.team, g)
		default:
		}
	}
}
