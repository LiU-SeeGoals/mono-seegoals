package demos

import (
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/scripting"
	"github.com/LiU-SeeGoals/controller/internal/simulator"
)

func RunScript(team info.Team, fn func(*scripting.Script)) {
	gameInfo := info.NewGameInfo(10)
	client.StartGameViewerServer()

	sslClientTracked := client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
	var sslClientRaw *client.SSLClient
	var send func([]action.Action)
	var teleporter scripting.Teleporter

	if config.IsSimulated() {
		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddress())
		simClient := client.NewSimClient(simAddressForTeam(team), gameInfo)
		send = simClient.SendActions

		simController := simulator.NewSimControl()
		ourRobots := []int{1, 2, 3, 4, 5, 6}
		if team == info.Yellow {
			simController.SetPresentRobots(ourRobots, nil)
		} else {
			simController.SetPresentRobots(nil, ourRobots)
		}
		teleporter = simController
	} else {
		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddressReal())
		basestationClient := client.NewBaseStationClient(config.GetBasestationAddress())
		basestationClient.Init()
		send = basestationClient.SendActions
	}

	teamAI := ai.NewAi(team, scripting.New(team, teleporter, fn), ai.NewActivityExecutor())

	for {
		if !sslClientRaw.WaitForVision(gameInfo) {
			send(stopAllRobots())
			continue
		}
		sslClientTracked.UpdateState(gameInfo, time.Now().UnixMilli())
		if !gameInfo.HasField() || !gameInfo.State.IsValid() {
			continue
		}

		actions := teamAI.GetActions(gameInfo)
		client.BroadcastActions(actions)
		send(actions)
	}
}
