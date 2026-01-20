package demos

import (
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	plan "github.com/LiU-SeeGoals/controller/internal/ai/plan"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

func FwRealScenario() {
	gameInfo := info.NewGameInfo(10)
	ssl_receiver_tracked_vision := client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
	ssl_receiver_raw_vision := client.NewSSLClient(config.GetSSLClientAddressReal())

	// Yellow team
	// slowBrainYellow := plan.NewPlannerFw(info.Yellow)
	slowBrainYellow := plan.NewPass(info.Yellow)
	slowBrainBlue := plan.NewPlannerGoalie(info.Blue)

	fastBrainYellow := ai.NewActivityExecutor()
	fastBrainBlue := ai.NewActivityExecutor()

	aiYellow := ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)
	aiBlue := ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

	basestationClient := client.NewBaseStationClient(config.GetBasestationAddress())

	basestationClient.Init()

	for {
		playTime := time.Now().UnixMilli()

		ssl_receiver_tracked_vision.UpdateState(gameInfo, playTime)
		ssl_receiver_raw_vision.UpdateState(gameInfo, playTime)
		yellow_actions := aiYellow.GetActions(gameInfo)
		blue_actions := aiBlue.GetActions(gameInfo)

		client.BroadcastActions(yellow_actions)
		client.BroadcastActions(blue_actions)

		basestationClient.SendActions(yellow_actions)
		basestationClient.SendActions(blue_actions)
	}
}
