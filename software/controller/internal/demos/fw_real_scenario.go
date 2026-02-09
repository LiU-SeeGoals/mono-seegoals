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

func handleSimulatedBall(gameInfo *info.GameInfo, simController *simulator.SimControl){

	ball := gameInfo.State.GetBall()
	ballPos, ballTime, _ := ball.GetPositionTime()
	if ballPos.Y > 3000 || ballPos.Y < -3000 || ballPos.X > 4500 || ballPos.X < -4500 || time.Now().UnixMilli()-ballTime > 5000 {
		simController.TeleportBall(0, 0)
	}

	ge := gameInfo.Status.GetGameEvent()
	previousState := ge.GetPreviousState()
	currentState := ge.GetCurrentState()
	switch currentState {
	case info.STATE_KICKOFF_PREPARATION:
		if previousState == info.STATE_HALTED || previousState == info.STATE_STOPPED {
			fmt.Println("teleported ball (new kickoff)")
			simController.TeleportBall(0, 0)
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
		}
	default:
	}
}

func FwRealScenario() {
	gameInfo := info.NewGameInfo(10)

	// var sslClientTracked *client.SSLTrackedClient
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
		teamBlue := []int{7}

		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddress())

		simClientYellow = client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)
		simClientBlue = client.NewSimClient(config.GetSimBlueTeamAddress(), gameInfo)

		simController = simulator.NewSimControl()
		simController.SetPresentRobots(teamYellow, teamBlue)

	} else {
		//sslClientTracked = client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddressReal())

		basestationClient = client.NewBaseStationClient(config.GetBasestationAddress())
		basestationClient.Init()
	}

	for {
		playTime := time.Now().UnixMilli()

		sslClientRaw.UpdateState(gameInfo, playTime)
		//sslClientTracked.UpdateState(gameInfo, playTime)

		actionsYellow := aiYellow.GetActions(gameInfo)
		actionsBlue := aiBlue.GetActions(gameInfo)

		client.BroadcastActions(actionsYellow)
		client.BroadcastActions(actionsBlue)

		if config.IsSimulated() {
			simClientYellow.SendActions(actionsYellow)
			simClientBlue.SendActions(actionsBlue)

			handleSimulatedBall(gameInfo, simController)

			time.Sleep(2 * time.Millisecond)
		} else {
			basestationClient.SendActions(actionsYellow)
			basestationClient.SendActions(actionsBlue)
		}
	}
}
