package demos

import (
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	plan "github.com/LiU-SeeGoals/controller/internal/ai/plan"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/simulator"
)

func FwRealScenario() {
	gameInfo := info.NewGameInfo(10)
	teamYellow := []int{1, 3}
	teamBlue := []int{7}

	//sslClientTracked := client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
	sslClientRaw := client.NewSSLClient(config.GetSSLClientAddress())

	simClientYellow := client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)
	simClientBlue := client.NewSimClient(config.GetSimBlueTeamAddress(), gameInfo)

	simController := simulator.NewSimControl()
	simController.SetPresentRobots(teamYellow, teamBlue)

	// Yellow team
	slowBrainYellow := plan.NewGameScenario(info.Yellow)
	slowBrainBlue := plan.NewPlannerGoalie(info.Blue)

	fastBrainYellow := ai.NewActivityExecutor()
	fastBrainBlue := ai.NewActivityExecutor()

	aiYellow := ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)
	aiBlue := ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

	basestationClient := client.NewBaseStationClient(config.GetBasestationAddress())
	basestationClient.Init()

	simController.TeleportRobot(-2000, 0, 1, info.Yellow)
  	simController.TeleportRobot(-2000, 500, 3, info.Yellow)
  	simController.TeleportRobot(1500, 0, 7, info.Blue)

	for {
		playTime := time.Now().UnixMilli()

		sslClientRaw.UpdateState(gameInfo, playTime)
		//sslClientTracked.UpdateState(gameInfo, playTime)

		actionsYellow := aiYellow.GetActions(gameInfo)
		actionsBlue := aiBlue.GetActions(gameInfo)

		client.BroadcastActions(actionsYellow)
		client.BroadcastActions(actionsBlue)

		basestationClient.SendActions(actionsYellow)
		basestationClient.SendActions(actionsBlue)

        simClientYellow.SendActions(actionsYellow)
        simClientBlue.SendActions(actionsBlue)

		time.Sleep(10 * time.Millisecond)
	}
}
