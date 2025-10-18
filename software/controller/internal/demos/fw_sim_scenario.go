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

func FwSimScenario() {
	presentYellow := []int{0, 1}
	presentBlue := []int{}

	simController := simulator.NewSimControl()
	simController.SetPresentRobots(presentYellow, presentBlue)

	gameInfo := info.NewGameInfo(10)
	ssl_receiver := client.NewSSLClient(config.GetSSLClientAddress())

	// Yellow team
	slowBrainYellow := plan.NewPlannerFw(info.Yellow)
	fastBrainYellow := ai.NewActivityExecutor()

	aiYellow := ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)

	basestationClient := client.NewBaseStationClient(config.GetBasestationAddress())
	simClient := client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)
    fmt.Println("Basedstation: ", config.GetBasestationAddress())

	basestationClient.Init()

	for {
		playTime := time.Now().UnixMilli()

		ssl_receiver.UpdateState(gameInfo, playTime)
		yellow_actions := aiYellow.GetActions(gameInfo)

		basestationClient.SendActions(yellow_actions)
        simClient.SendActions(yellow_actions)
	}
}
