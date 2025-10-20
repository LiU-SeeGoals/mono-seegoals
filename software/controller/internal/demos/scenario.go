package demos

import (
	"fmt"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/ai"
	plan "github.com/LiU-SeeGoals/controller/internal/ai/plan"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/simulator"
	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"google.golang.org/protobuf/proto"
)

const MAX_SEND_SIZE = 2048

func Scenario() {
	presentYellow := []int{0, 1, 2, 3, 4}
	presentBlue := []int{0, 1, 2, 3, 4}

	simController := simulator.NewSimControl()
	simController.SetPresentRobots(presentYellow, presentBlue)

	gameInfo := info.NewGameInfo(10)
	ssl_receiver := client.NewSSLClient(config.GetSSLClientAddress())

	// Yellow team
	slowBrainYellow := plan.NewPlanner1(info.Yellow)
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

		client.BroadcastActions(yellow_actions) // We broadcast actions for the GV to print 'em
		basestationClient.SendActions(yellow_actions)
		simClient.SendActions(yellow_actions)
	}
}

func testPacketSize(actions []action.Action) {
	var queue []*robot_action.Command

	for _, action := range actions {
		queue = append(queue, action.TranslateReal())
	}

	for _, cmd := range queue {

		serializedCmd, _ := proto.Marshal(cmd) // Add error handling
		if len(serializedCmd) > MAX_SEND_SIZE {
			fmt.Print("to big to send (if sent = Rasmus mad 😡)")
		} else {
			fmt.Println("Packet size: ", len(serializedCmd))
		}

	}
}
