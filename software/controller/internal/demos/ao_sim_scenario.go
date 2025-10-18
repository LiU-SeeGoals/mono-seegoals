package demos

// import (
// 	// "fmt"
// 	"time"

// 	"github.com/LiU-SeeGoals/controller/internal/ai"
// 	slow_brain "github.com/LiU-SeeGoals/controller/internal/ai/slow_brain"
// 	"github.com/LiU-SeeGoals/controller/internal/client"
// 	"github.com/LiU-SeeGoals/controller/internal/config"
// 	"github.com/LiU-SeeGoals/controller/internal/info"
// 	"github.com/LiU-SeeGoals/controller/internal/simulator"
// )

// func AoSimScenario() {
// 	presentYellow := []int{0, 1, 3}
// 	presentBlue := []int{0, 1, 3}

// 	simController := simulator.NewSimControl()
// 	simController.SetPresentRobots(presentYellow, presentBlue)

// 	gameInfo := info.NewGameInfo(10)
// 	ssl_receiver := client.NewSSLClient(config.GetSSLClientAddress())

// 	// Yellow team
// 	slowBrainYellow := slow_brain.NewPlannerAo(info.Yellow)
// 	fastBrainYellow := ai.NewFastBrainGO()

// 	// Blue team
// 	slowBrainBlue := slow_brain.NewPlannerAo(info.Blue)
// 	fastBrainBlue := ai.NewFastBrainGO()

// 	aiYellow := ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)

// 	aiBlue := ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

// 	// basestationClient := client.NewBaseStationClient(config.GetBasestationAddress())
// 	simClientYellow := client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)
// 	simClientBlue := client.NewSimClient(config.GetSimBlueTeamAddress(), gameInfo)
//     // fmt.Println("Basedstation: ", config.GetBasestationAddress())

// 	// basestationClient.Init()

// 	for {
// 		playTime := time.Now().UnixMilli()

// 		ssl_receiver.UpdateState(gameInfo, playTime)

// 		yellow_actions := aiYellow.GetActions(gameInfo)
//         simClientYellow.SendActions(yellow_actions)

// 		blue_actions := aiBlue.GetActions(gameInfo)
//         simClientBlue.SendActions(blue_actions)
// 	}
// }
