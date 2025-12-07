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

/*
 * This scenario is for demonstrating/testing the GameViewer website.
 */

func GameViewerDemo() {
	teamYellow := []int{0, 1, 2}
	teamBlue := []int{0, 1, 2}

	gameInfo := info.NewGameInfo(10)

	sslClient := client.NewSSLClient(config.GetSSLClientAddress())
	simClientYellow := client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)
	simClientBlue := client.NewSimClient(config.GetSimBlueTeamAddress(), gameInfo)

	simController := simulator.NewSimControl()
	simController.SetPresentRobots(teamYellow, teamBlue)

	slowBrainYellow := plan.NewPlannerGWDemo(info.Yellow)
	fastBrainYellow := ai.NewActivityExecutor()

	slowBrainBlue := plan.NewPlannerGWDemo(info.Blue)
	fastBrainBlue := ai.NewActivityExecutor()

	aiYellow := ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)
	aiBlue := ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

	//basestationClient := client.NewBaseStationClient(config.GetBasestationAddress())
	//fmt.Println("Base(d)station: ", config.GetBasestationAddress())
	//basestationClient.Init()

	for {
		playTime := time.Now().UnixMilli()

		sslClient.UpdateState(gameInfo, playTime)
		actionsYellow := aiYellow.GetActions(gameInfo)
		actionsBlue := aiBlue.GetActions(gameInfo)

		// We broadcast actions for the GV to print 'em
        client.BroadcastActions(actionsYellow)
        client.BroadcastActions(actionsBlue)

		//basestationClient.SendActions(yellow_actions)
        simClientYellow.SendActions(actionsYellow)
        simClientBlue.SendActions(actionsBlue)
	}
}
