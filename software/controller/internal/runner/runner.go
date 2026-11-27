// Package runner owns the controller's main loop: it wires vision, referee,
// planners and robot outputs together and steps them once per vision frame.
// Every scenario (match, vs-TIGERs, real hardware, scripts) is a Config.
package runner

import (
	"fmt"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/ai/plan"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/scripting"
	"github.com/LiU-SeeGoals/controller/internal/simulator"
)

// Config describes one controller run.
type Config struct {
	// Teams controlled by this process.
	Teams []info.Team
	// Simulated selects simulator addresses and sim command output; otherwise
	// real vision addresses and the basestation are used.
	Simulated bool
	// Planner builds a team's planner. sim is nil on real hardware.
	Planner func(team info.Team, sim *simulator.SimControl) ai.Planner
	// ManualPlanner, when set, lets the GameViewer's CHANGE_SCENARIO
	// "Manual"/"Game" commands swap between it and Planner.
	ManualPlanner func(team info.Team) ai.Planner
	// PresentYellow/PresentBlue are the robot IDs the simulator spawns.
	PresentYellow, PresentBlue []int
	// ManageBall lets the runner act as ball placer in simulation.
	ManageBall bool
}

var simTeam = []int{1, 2, 3, 4, 5, 6}

// Match is a normal game for the given teams with the referee-aware planner.
// In simulation both teams are spawned and the runner places the ball.
func Match(simulated bool, teams ...info.Team) Config {
	return Config{
		Teams:     teams,
		Simulated: simulated,
		Planner: func(team info.Team, _ *simulator.SimControl) ai.Planner {
			return plan.NewCombinedPlanWithRef(team)
		},
		ManualPlanner: func(team info.Team) ai.Planner {
			return plan.NewPlannerManualMovement(team)
		},
		PresentYellow: simTeam,
		PresentBlue:   simTeam,
		ManageBall:    true,
	}
}

// Script runs a scripting.Script for one team. In simulation only that team is
// spawned and the script may teleport robots and the ball.
func Script(simulated bool, team info.Team, fn func(*scripting.Script)) Config {
	cfg := Config{
		Teams:     []info.Team{team},
		Simulated: simulated,
		Planner: func(team info.Team, sim *simulator.SimControl) ai.Planner {
			var tp scripting.Teleporter
			if sim != nil { // avoid a non-nil interface holding a nil pointer
				tp = sim
			}
			return scripting.New(team, tp, fn)
		},
	}
	if team == info.Yellow {
		cfg.PresentYellow = simTeam
	} else {
		cfg.PresentBlue = simTeam
	}
	return cfg
}

type teamRunner struct {
	team   info.Team
	ai     *ai.Ai
	game   ai.Planner
	manual ai.Planner
	send   func([]action.Action)
}

// Run blocks forever, stepping the AI once per raw vision frame.
func Run(cfg Config) {
	if len(cfg.Teams) == 0 || cfg.Planner == nil {
		panic("runner: Config needs Teams and Planner")
	}

	gameInfo := info.NewGameInfo(10)
	client.StartGameViewerServer()

	sslClientTracked := client.NewSSLTrackedClient(config.GetSSLTrackedClientAddress())
	var sslClientRaw *client.SSLClient
	var simController *simulator.SimControl
	var basestation *client.BaseStationClient

	if cfg.Simulated {
		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddress())
		simController = simulator.NewSimControl()
		if cfg.ManageBall {
			teleportBallMillimeters(simController, info.Position{})
		}
		simController.SetPresentRobots(cfg.PresentYellow, cfg.PresentBlue)
	} else {
		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddressReal())
		basestation = client.NewBaseStationClient(config.GetBasestationMulticastAddress())
		basestation.Init()
	}

	teams := make([]*teamRunner, 0, len(cfg.Teams))
	for _, team := range cfg.Teams {
		tr := &teamRunner{team: team, game: cfg.Planner(team, simController)}
		if cfg.ManualPlanner != nil {
			tr.manual = cfg.ManualPlanner(team)
		}
		tr.ai = ai.NewAi(team, tr.game)
		if cfg.Simulated {
			tr.send = client.NewSimClient(simAddressForTeam(team)).SendActions
		} else {
			tr.send = basestation.SendActions
		}
		teams = append(teams, tr)
	}

	ballHandler := simulatedBallHandler{}
	for {
		// Pace the control loop from raw vision frames.
		if !sslClientRaw.WaitForVision(gameInfo) {
			for _, tr := range teams {
				tr.send(stopAllRobots())
			}
			continue
		}

		sslClientTracked.UpdateState(gameInfo, time.Now().UnixMilli())

		if command := client.GetCommand(client.CHANGE_SCENARIO); command != nil {
			for _, tr := range teams {
				if tr.manual == nil {
					continue
				}
				switch command.Type {
				case "Manual":
					tr.ai.HotswapPlanner(tr.team, tr.manual)
				case "Game":
					tr.ai.HotswapPlanner(tr.team, tr.game)
				}
			}
		}

		if !gameInfo.HasField() || !gameInfo.State.IsValid() {
			continue
		}

		for _, tr := range teams {
			actions := tr.ai.GetActions(gameInfo)
			client.BroadcastActions(actions)
			tr.send(actions)
		}

		if cfg.Simulated && cfg.ManageBall {
			ballHandler.handle(gameInfo, simController)
		}
	}
}

func simAddressForTeam(team info.Team) string {
	switch team {
	case info.Blue:
		return config.GetSimBlueTeamAddress()
	case info.Yellow:
		return config.GetSimYellowTeamAddress()
	default:
		panic(fmt.Sprintf("unsupported team: %s", team))
	}
}

// Stop every ID, including robots absent from the last vision frame.
func stopAllRobots() []action.Action {
	actions := make([]action.Action, 0, info.TEAM_SIZE)
	for id := info.ID(0); id < info.TEAM_SIZE; id++ {
		actions = append(actions, &action.Stop{Id: int(id)})
	}
	return actions
}
