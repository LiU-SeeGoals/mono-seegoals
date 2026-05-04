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

// handleDefenceBall resets the ball to center whenever Yellow loses it.
// This keeps the scenario focused on defense: Yellow always attacks, Blue always defends.
//
// Reset triggers:
//   - Ball goes out of bounds
//   - Ball has not been detected for > 5 seconds (stuck / lost)
//   - Blue robot gains possession — defense succeeded, restart from center
func handleDefenceBall(gameInfo *info.GameInfo, simController *simulator.SimControl) {
	ball := gameInfo.State.GetBall()
	ballPos, ballTime, _ := ball.GetPositionTime()

	outOfBounds := ballPos.X > 4500 || ballPos.X < -4500 ||
		ballPos.Y > 3000 || ballPos.Y < -3000

	stale := time.Now().UnixMilli()-ballTime > 5000

	// Blue gained possession — defense succeeded, reset so Yellow can attack again
	possessor := ball.GetPossessor()
	blueGainedPossession := possessor != nil && possessor.GetTeam() == info.Blue

	if outOfBounds || stale || blueGainedPossession {
		if blueGainedPossession {
			fmt.Println("Blue gained possession — resetting ball to center")
		}
		simController.TeleportBall(0, 0)
	}
}

func Defence2BotScenario() {
	gameInfo := info.NewGameInfo(10)
	client.StartGameViewerServer()

	var sslClientTracked *client.SSLTrackedClient
	var sslClientRaw *client.SSLClient
	var aiYellow *ai.Ai
	var aiBlueDefense *ai.Ai
	var basestationClient *client.BaseStationClient
	var simController *simulator.SimControl
	var simClientYellow *client.SimClient
	var simClientBlue *client.SimClient

	// Yellow: 3-bot offense (robots 1, 2, 3)
	aiYellow = ai.NewAi(info.Yellow, plan.NewGameScenario(info.Yellow), ai.NewActivityExecutor())

	// Blue: 3-bot defense wall (robots 5, 6, 7) — no goalie in this scenario
	// Bot 5 = presser, bot 6 = deep defender, bot 7 = second defender
	aiBlueDefense = ai.NewAi(info.Blue, plan.NewTwoBotDefence(info.Blue, []info.ID{5, 6, 7, 8, 9}), ai.NewActivityExecutor())

	if config.IsSimulated() {
		teamYellow := []int{1, 2, 3, 4, 10}
		teamBlue := []int{5, 6, 7, 8, 9}

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
		actionsBlue := aiBlueDefense.GetActions(gameInfo)

		client.BroadcastActions(actionsYellow)
		client.BroadcastActions(actionsBlue)

		if config.IsSimulated() {
			simClientYellow.SendActions(actionsYellow)
			simClientBlue.SendActions(actionsBlue)

			handleDefenceBall(gameInfo, simController)
		} else {
			basestationClient.SendActions(actionsYellow)
			basestationClient.SendActions(actionsBlue)
		}
	}
}
