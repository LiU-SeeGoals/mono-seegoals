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

const MAX_SEND_SIZE_DEFENCE = 2048

// ScenarioDefencePlan runs a copy of Scenario but using PlannerDefence instead of Planner1.
func ScenarioDefencePlan() {
	presentYellow := []int{0, 1, 2, 3, 4}
	presentBlue := []int{0, 1, 2, 3, 4}

	simController := simulator.NewSimControl()
	simController.SetPresentRobots(presentYellow, presentBlue)

	gameInfo := info.NewGameInfo(10)
	sslReceiver := client.NewSSLClient(config.GetSSLClientAddress())

	// Yellow team
	slowBrainYellow := plan.NewPlannerDefence(info.Yellow)
	fastBrainYellow := ai.NewActivityExecutor()
	aiYellow := ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)

	// Blue team
	slowBrainBlue := plan.NewPlannerDefence(info.Blue)
	fastBrainBlue := ai.NewActivityExecutor()
	aiBlue := ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

	simClientYellow := client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)
	simClientBlue := client.NewSimClient(config.GetSimBlueTeamAddress(), gameInfo)

	for {
		playTime := time.Now().UnixMilli()

		sslReceiver.UpdateState(gameInfo, playTime)
		yellowActions := aiYellow.GetActions(gameInfo)
		blueActions := aiBlue.GetActions(gameInfo)

		allActions := append([]action.Action{}, yellowActions...)
		allActions = append(allActions, blueActions...)

		client.BroadcastActions(allActions)
		simClientYellow.SendActions(yellowActions)
		simClientBlue.SendActions(blueActions)
	}
}

func testPacketSizeDefence(actions []action.Action) {
	var queue []*robot_action.Command

	for _, action := range actions {
		queue = append(queue, action.TranslateReal())
	}

	for _, cmd := range queue {

		serializedCmd, _ := proto.Marshal(cmd) // Add error handling
		if len(serializedCmd) > MAX_SEND_SIZE_DEFENCE {
			fmt.Print("to big to send (if sent = Rasmus mad 😡)")
		} else {
			fmt.Println("Packet size: ", len(serializedCmd))
		}

	}
}
