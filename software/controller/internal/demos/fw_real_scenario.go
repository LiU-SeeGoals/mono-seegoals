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
	sslClientTracked := client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
	sslClientRaw := client.NewSSLClient(config.GetSSLClientAddressReal())

	slowBrainYellow := plan.NewGameScenario(info.Yellow)
	slowBrainBlue := plan.NewPlannerGoalie(info.Blue)

	fastBrainYellow := ai.NewActivityExecutor()
	fastBrainBlue := ai.NewActivityExecutor()

	aiYellow := ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)
	aiBlue := ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

	basestationClient := client.NewBaseStationClient(config.GetBasestationAddress())
	basestationClient.Init()

	if config.IsSimulated() {
		//teamYellow := []int{1, 3}
		//teamBlue := []int{7}

		//sslClientRaw := client.NewSSLClient(config.GetSSLClientAddress())

		//simClientYellow := client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)
		//simClientBlue := client.NewSimClient(config.GetSimBlueTeamAddress(), gameInfo)

		//simController := simulator.NewSimControl()
		//simController.SetPresentRobots(teamYellow, teamBlue)

		//simController.TeleportRobot(-2000, 0, 1, info.Yellow)
		//simController.TeleportRobot(-10000, 500, 3, info.Yellow)
		//simController.TeleportRobot(1500, 0, 7, info.Blue)
	} else {
		//sslClientTracked = client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
		//sslClientRaw = client.NewSSLClient(config.GetSSLClientAddressReal())
		//
		//slowBrainYellow := plan.NewGameScenario(info.Yellow)
		//slowBrainBlue := plan.NewPlannerGoalie(info.Blue)
		//
		//fastBrainYellow := ai.NewActivityExecutor()
		//fastBrainBlue := ai.NewActivityExecutor()
		//
		//aiYellow = ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)
		//aiBlue = ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)
		//
		//basestationClient = client.NewBaseStationClient(config.GetBasestationAddress())
		//basestationClient.Init()
	}

	for {
		playTime := time.Now().UnixMilli()

		sslClientRaw.UpdateState(gameInfo, playTime)
		sslClientTracked.UpdateState(gameInfo, playTime)

		// *** REAL ***
		if config.IsSimulated() {
		} else {
			actionsYellow := aiYellow.GetActions(gameInfo)
			actionsBlue := aiBlue.GetActions(gameInfo)

			client.BroadcastActions(actionsYellow)
			client.BroadcastActions(actionsBlue)

			basestationClient.SendActions(actionsYellow)
			basestationClient.SendActions(actionsBlue)
		}

		// *** SIM ***
		//ball, _ := gameInfo.State.GetBall().GetPosition()
		// If the ball is outside the field put it back
		//if math.Abs(ball.Y) > 3000 || math.Abs(ball.X) > 4500 {
		//	simController.TeleportBall(0, 0)
		//}

		//simClientYellow.SendActions(actionsYellow)
		//simClientBlue.SendActions(actionsBlue)

		//time.Sleep(10 * time.Millisecond)
	}
}
