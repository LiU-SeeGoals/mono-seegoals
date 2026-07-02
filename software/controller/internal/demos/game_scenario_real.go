package demos

import (
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	plan "github.com/LiU-SeeGoals/controller/internal/ai/plan"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func gameScenarioReal(team info.Team) {
	gameInfo := info.NewGameInfo(10)
	client.StartGameViewerServer()

	sslClientTracked := client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
	sslClientRaw := client.NewSSLClient(config.GetSSLClientAddressReal())

	slowBrain := plan.NewCombinedPlanWithRef(team)
	manualMovement := plan.NewPlannerManualMovement(team)
	fastBrain := ai.NewActivityExecutor()
	teamAI := ai.NewAi(team, slowBrain, fastBrain)

	basestationClient := client.NewBaseStationClient(config.GetBasestationAddress())
	basestationClient.Init()

	for {
		// Pace the control loop from raw vision frames.
		sslClientRaw.WaitForVision(gameInfo)

		playTime := time.Now().UnixMilli()
		sslClientTracked.UpdateState(gameInfo, playTime)

		command := client.GetCommand(client.CHANGE_SCENARIO)
		if command != nil {
			if command.Type == "Manual" {
				teamAI.HotswapPlanner(team, manualMovement)
			} else if command.Type == "Game" {
				teamAI.HotswapPlanner(team, slowBrain)
			}
		}

		if !gameInfo.HasField() || !gameInfo.State.IsValid() {
			continue
		}

		actions := teamAI.GetActions(gameInfo)
		client.BroadcastActions(actions)
		basestationClient.SendActions(actions)
	}
}
