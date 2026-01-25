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

	//var sslClientTracked *client.SSLTrackedClient
	var sslClientRaw *client.SSLClient
	var aiYellow *ai.Ai
	var aiBlue *ai.Ai
	var basestationClient *client.BaseStationClient
	var simController *simulator.SimControl
	var simClientYellow *client.SimClient
	var simClientBlue *client.SimClient

	if config.IsSimulated() {
		teamYellow := []int{1, 3}
		teamBlue := []int{7, 1, 2, 3, 4}

		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddress())

		simClientYellow = client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)
		simClientBlue = client.NewSimClient(config.GetSimBlueTeamAddress(), gameInfo)

		simController = simulator.NewSimControl()
		simController.SetPresentRobots(teamYellow, teamBlue)

		slowBrainYellow := plan.NewGameScenario(info.Yellow)
		slowBrainBlue := plan.NewPlannerGoalie(info.Blue)

		fastBrainYellow := ai.NewActivityExecutor()
		fastBrainBlue := ai.NewActivityExecutor()

		aiYellow = ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)
		aiBlue = ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

		simController.TeleportRobot(-1000, 0, 1, info.Blue)
		simController.TeleportRobot(1000, 700, 2, info.Blue)
		simController.TeleportRobot(1500, 100, 3, info.Blue)

		simController.TeleportRobot(-2000, 0, 1, info.Yellow)
		simController.TeleportRobot(1000, 500, 3, info.Yellow)
		simController.TeleportRobot(1500, 0, 7, info.Blue)
	} else {
		//sslClientTracked = client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddressReal())

		slowBrainYellow := plan.NewGameScenario(info.Yellow)
		slowBrainBlue := plan.NewPlannerGoalie(info.Blue)

		fastBrainYellow := ai.NewActivityExecutor()
		fastBrainBlue := ai.NewActivityExecutor()

		aiYellow = ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)
		aiBlue = ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

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

			ball := gameInfo.State.GetBall()
			ballPos, ballTime, _ := ball.GetPositionTime()
			if ballPos.Y > 3000 || ballPos.Y < -3000 || ballPos.X > 4500 || ballPos.X < -4500 || time.Now().UnixMilli()-ballTime > 5000 {
				simController.TeleportBall(0, 0)
			}

			time.Sleep(2 * time.Millisecond)
		} else {
			basestationClient.SendActions(actionsYellow)
			basestationClient.SendActions(actionsBlue)
		}
	}
}
