package demos

import (
	"fmt"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	plan "github.com/LiU-SeeGoals/controller/internal/ai/plan"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/simulator"
)

const (
	ballStillnessTimeoutMs = 10000
	ballStillnessRadius    = 20.0
)

var (
	lastMovingBallPos   info.Position
	lastMovingBallTime  int64
	hasMovingBallSample bool
)

func resetBallStillnessTracker(pos info.Position, now int64) {
	lastMovingBallPos = pos
	lastMovingBallTime = now
	hasMovingBallSample = true
}

func handleSimulatedBall(gameInfo *info.GameInfo, simController *simulator.SimControl) {
	ball := gameInfo.State.GetBall()
	ballPos, ballTime, _ := ball.GetPositionTime()
	now := time.Now().UnixMilli()

	if !hasMovingBallSample {
		resetBallStillnessTracker(ballPos, now)
	} else if ballPos.Dist2d(lastMovingBallPos) > ballStillnessRadius {
		resetBallStillnessTracker(ballPos, now)
	}

	if ballPos.Y > 3000 ||
		ballPos.Y < -3000 ||
		ballPos.X > 4500 ||
		ballPos.X < -4500 ||
		now-ballTime > 5000 ||
		now-lastMovingBallTime > ballStillnessTimeoutMs {
		simController.TeleportBall(0, 0)
		resetBallStillnessTracker(info.Position{X: 0, Y: 0, Z: 0, Angle: 0}, now)
	}

	ge := gameInfo.Status.GetGameEvent()
	previousState := ge.GetPreviousState()
	currentState := ge.GetCurrentState()
	switch currentState {
	case info.STATE_KICKOFF_PREPARATION:
		if previousState == info.STATE_HALTED || previousState == info.STATE_STOPPED {
			fmt.Println("teleported ball (new kickoff)")
			simController.TeleportBall(0, 0)
			resetBallStillnessTracker(info.Position{X: 0, Y: 0, Z: 0, Angle: 0}, now)
		}
	case info.STATE_FREE_KICK:
	case info.STATE_HALTED, info.STATE_STOPPED:
	case info.STATE_PENALTY_PREPARATION, info.STATE_TIMEOUT:
	case info.STATE_PLAYING:
	case info.STATE_BALL_PLACEMENT:
		ball, _ := gameInfo.State.GetBall().GetPosition()
		if previousState != info.STATE_BALL_PLACEMENT {
			toX := ge.GetDesignatedPosition().At(0, 0)
			toY := ge.GetDesignatedPosition().At(1, 0)
			finalX := ball.X - toX
			finalY := ball.Y - toY
			fmt.Printf("teleported ball (%f, %f) (ball placement %s)\n", toX, toY, ge.GetTeamWithPossession())
			simController.TeleportBall(float32(finalX), float32(finalY))
			resetBallStillnessTracker(info.Position{X: toX, Y: toY, Z: 0, Angle: 0}, now)
		}
	default:
	}
}

func FwRealScenario() {
	gameInfo := info.NewGameInfo(10)

	var sslClientTracked *client.SSLTrackedClient
	var sslClientRaw *client.SSLClient
	var aiYellow *ai.Ai
	var aiBlue *ai.Ai
	var basestationClient *client.BaseStationClient
	var simController *simulator.SimControl
	var simClientYellow *client.SimClient
	var simClientBlue *client.SimClient

	slowBrainYellow := plan.NewGameScenario(info.Yellow)
	slowBrainBlue := plan.NewPlannerGoalie(info.Blue)
	//slowBrainBlue := plan.NewGameScenario2(info.Blue)

	fastBrainYellow := ai.NewActivityExecutor()
	fastBrainBlue := ai.NewActivityExecutor()

	aiYellow = ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)
	aiBlue = ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

	if config.IsSimulated() {
		teamYellow := []int{1, 3}
		teamBlue := []int{1, 7}

		sslClientTracked = client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddress())

		simClientYellow = client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)
		simClientBlue = client.NewSimClient(config.GetSimBlueTeamAddress(), gameInfo)

		simController = simulator.NewSimControl()
		simController.SetPresentRobots(teamYellow, teamBlue)
	} else {
		sslClientTracked = client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddressReal())

		basestationClient = client.NewBaseStationClient(config.GetBasestationAddress())
		basestationClient.Init()
	}

	for {
		playTime := time.Now().UnixMilli()

		sslClientRaw.UpdateState(gameInfo, playTime)
		sslClientTracked.UpdateState(gameInfo, playTime)

		if !gameInfo.HasField() || !gameInfo.State.IsValid() {
			continue
		}

		actionsYellow := aiYellow.GetActions(gameInfo)
		actionsBlue := aiBlue.GetActions(gameInfo)

		client.BroadcastActions(actionsYellow)
		client.BroadcastActions(actionsBlue)

		if config.IsSimulated() {
			simClientYellow.SendActions(actionsYellow)
			simClientBlue.SendActions(actionsBlue)

			handleSimulatedBall(gameInfo, simController)
		} else {
			basestationClient.SendActions(actionsYellow)
			basestationClient.SendActions(actionsBlue)
		}
	}
}
