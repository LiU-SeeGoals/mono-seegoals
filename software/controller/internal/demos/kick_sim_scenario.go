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

func KickSimScenario() {
	// This avoid the "No position in history" error for robots
	presentYellow := []int{1, 3}
	presentBlue := []int{7}
	simController := simulator.NewSimControl()
	simController.SetPresentRobots(presentYellow, presentBlue)

	gameInfo := info.NewGameInfo(10)
	ssl_receiver := client.NewSSLClient(config.GetSSLClientAddress())

	TestKickPlan := plan.NewPass(info.Yellow)
	Executor := ai.NewActivityExecutor()

	aiYellow := ai.NewAi(info.Yellow, TestKickPlan, Executor)

	simClientYellow := client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)
	simController.SetPresentRobots(presentYellow, presentBlue)

	for {
		playTime := time.Now().UnixMilli()
		ssl_receiver.UpdateState(gameInfo, playTime)

		yellow_actions := aiYellow.GetActions(gameInfo)
		simClientYellow.SendActions(yellow_actions)
	}
}
