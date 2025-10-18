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

func Goalie() {
	// This avoid the "No position in history" error for robots
	presentYellow := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	presentBlue := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	simController := simulator.NewSimControl()
	simController.SetPresentRobots(presentYellow, presentBlue)

	gameInfo := info.NewGameInfo(10)
	ssl_receiver := client.NewSSLClient(config.GetSSLClientAddress())

	// Yellow team
	slowBrainYellow := plan.NewPlanner1(info.Yellow)
	fastBrainYellow := ai.NewActivityExecutor()

	aiYellow := ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)

	simClientYellow := client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)

	// Blue team
	// slowBrainBlue := slow_brain.NewPlanner1(info.Blue)
	// fastBrainBlue := ai.NewFastBrainGO()

	// aiBlue := ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

	// simClientBlue := client.NewSimClient(config.GetSimBlueTeamAddress(), gameInfo)

	// Some sim setup for debugging ai behaviour
	presentYellow = []int{0, 1, 2, 3, 4, 5,6 , 7 , 8, 9 ,10}
	presentBlue = []int{0, 1, 2, 3, 4, 5, 6 ,7, 8, 9 ,10}
	simController.SetPresentRobots(presentYellow, presentBlue)

	start_time := time.Now().UnixMilli()
	for {
		playTime := time.Now().UnixMilli() - start_time
		// fmt.Println("playTime: ", playTime)
		ssl_receiver.UpdateState(gameInfo, playTime)
		//fmt.Println(gameInfo)

		yellow_actions := aiYellow.GetActions(gameInfo)
		simClientYellow.SendActions(yellow_actions)

		//blue_actions := aiBlue.GetActions(gameInfo)
		//simClientBlue.SendActions(blue_actions)

		// terminal_messages := []string{"Scenario"}

		// if len(blue_actions) > 0 {
		// 	client.UpdateWebGUI(gameInfo, blue_actions, terminal_messages)
		// }
	}
}
