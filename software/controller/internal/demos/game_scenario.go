package demos

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	plan "github.com/LiU-SeeGoals/controller/internal/ai/plan"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/simulator"
)

const simulatedGoalAreaTeleportInterval = 15 * time.Second

type goalAreaTeleporter struct {
	nextTeleport time.Time
	rng          *rand.Rand
}

func newGoalAreaTeleporter() *goalAreaTeleporter {
	return &goalAreaTeleporter{
		nextTeleport: time.Now().Add(simulatedGoalAreaTeleportInterval),
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (t *goalAreaTeleporter) MaybeTeleport(gameInfo *info.GameInfo, simController *simulator.SimControl) {
	if t == nil || simController == nil || time.Now().Before(t.nextTeleport) {
		return
	}

	x, y := randomGoalAreaPosition(gameInfo, t.rng)
	fmt.Printf("teleported ball (%f, %f) (periodic goal-area test)\n", x, y)
	simController.TeleportBall(float32(x/1000), float32(y/1000))
	t.nextTeleport = time.Now().Add(simulatedGoalAreaTeleportInterval)
}

func randomGoalAreaPosition(gameInfo *info.GameInfo, rng *rand.Rand) (float64, float64) {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	if gameInfo != nil && gameInfo.HasField() {
		if rng.Intn(2) == 0 {
			if x, y, ok := randomGoalAreaPositionFromLines(gameInfo, "LeftPenaltyStretch", "LeftGoalLine", rng); ok {
				return x, y
			}
		}
		if x, y, ok := randomGoalAreaPositionFromLines(gameInfo, "RightPenaltyStretch", "RightGoalLine", rng); ok {
			return x, y
		}
	}

	xSign := 1.0
	if rng.Intn(2) == 0 {
		xSign = -1.0
	}
	return randomBetween(3500, 4500, rng) * xSign, randomBetween(-1000, 1000, rng)
}

func randomGoalAreaPositionFromLines(gameInfo *info.GameInfo, frontLineName, backLineName string, rng *rand.Rand) (float64, float64, bool) {
	front := gameInfo.GetFieldLine(frontLineName)
	back := gameInfo.GetFieldLine(backLineName)
	if front == nil || back == nil || front.GetP1() == nil || front.GetP2() == nil || back.GetP1() == nil {
		return 0, 0, false
	}

	const margin = 150.0
	minX := math.Min(float64(front.GetP1().GetX()), float64(back.GetP1().GetX())) + margin
	maxX := math.Max(float64(front.GetP1().GetX()), float64(back.GetP1().GetX())) - margin
	minY := math.Min(float64(front.GetP1().GetY()), float64(front.GetP2().GetY())) + margin
	maxY := math.Max(float64(front.GetP1().GetY()), float64(front.GetP2().GetY())) - margin
	if minX >= maxX || minY >= maxY {
		return 0, 0, false
	}

	return randomBetween(minX, maxX, rng), randomBetween(minY, maxY, rng), true
}

func randomBetween(minValue, maxValue float64, rng *rand.Rand) float64 {
	return minValue + rng.Float64()*(maxValue-minValue)
}

func handleSimulatedBall(gameInfo *info.GameInfo, simController *simulator.SimControl, goalAreaTeleporter *goalAreaTeleporter) {

	ball := gameInfo.State.GetBall()
	ballPos, ballTime, _ := ball.GetPositionTime()
	if ballPos.Y > 3000 || ballPos.Y < -3000 || ballPos.X > 4500 || ballPos.X < -4500 || time.Now().UnixMilli()-ballTime > 5000 {
		simController.TeleportBall(0, 1)
	}

	ge := gameInfo.Status.GetGameEvent()
	previousState := ge.GetPreviousState()
	currentState := ge.GetCurrentState()
	switch currentState {
	case info.STATE_KICKOFF_PREPARATION:
		if previousState == info.STATE_HALTED || previousState == info.STATE_STOPPED {
			// fmt.Println("teleported ball (new kickoff)")
			simController.TeleportBall(0, 1000)
		}
	case info.STATE_FREE_KICK:
	case info.STATE_HALTED, info.STATE_STOPPED:
	case info.STATE_PENALTY_PREPARATION, info.STATE_TIMEOUT:
	case info.STATE_PLAYING:
	case info.STATE_BALL_PLACEMENT:
		ball, _ := gameInfo.State.GetBall().GetPosition()
		if previousState != info.STATE_BALL_PLACEMENT {
			toX := ge.GetDesignatedPosition().At(0, 0)
			toY := ge.GetDesignatedPosition().At(1, 0)
			finalX := ball.X - toX
			finalY := ball.Y - toY
			fmt.Printf("teleported ball (%f, %f) (ball placement %s)\n", toX, toY, ge.GetTeamWithPossession())
			simController.TeleportBall(float32(finalX), float32(finalY))
		}
	default:
	}

	goalAreaTeleporter.MaybeTeleport(gameInfo, simController)
}

func GameScenario() {
	gameInfo := info.NewGameInfo(10)
	client.StartGameViewerServer()

	var sslClientTracked *client.SSLTrackedClient
	var sslClientRaw *client.SSLClient
	var aiYellow *ai.Ai
	var aiBlue *ai.Ai
	var basestationClient *client.BaseStationClient
	var simController *simulator.SimControl
	var goalAreaTeleporter *goalAreaTeleporter
	var simClientYellow *client.SimClient
	var simClientBlue *client.SimClient

	slowBrainYellow := plan.NewCombinedPlanWithRef(info.Yellow)
	slowBrainBlue := plan.NewCombinedPlanWithRef(info.Blue)

	manualMovementYellow := plan.NewPlannerManualMovement(info.Yellow)
	// manualMovementBlue := plan.NewPlannerManualMovement(info.Blue)

	fastBrainYellow := ai.NewActivityExecutor()
	fastBrainBlue := ai.NewActivityExecutor()

	aiYellow = ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)
	aiBlue = ai.NewAi(info.Blue, slowBrainBlue, fastBrainBlue)

	if config.IsSimulated() {
		teamYellow := []int{1, 2, 3, 4, 5, 6}
		teamBlue := []int{1, 2, 3, 4, 5, 6}

		sslClientTracked = client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddress())

		simClientYellow = client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)
		simClientBlue = client.NewSimClient(config.GetSimBlueTeamAddress(), gameInfo)

		simController = simulator.NewSimControl()
		simController.TeleportBall(0, 1000)
		simController.SetPresentRobots(teamYellow, teamBlue)
		goalAreaTeleporter = newGoalAreaTeleporter()

	} else {
		sslClientTracked = client.NewSSLTrackedClient(config.GetSSLTrackedClientAddressReal())
		sslClientRaw = client.NewSSLClient(config.GetSSLClientAddressReal())

		basestationClient = client.NewBaseStationClient(config.GetBasestationAddress())
		basestationClient.Init()
	}
	for {
		// Pace the control loop from raw vision frames.
		sslClientRaw.WaitForVision(gameInfo)

		playTime := time.Now().UnixMilli()

		sslClientTracked.UpdateState(gameInfo, playTime)

		command := client.GetCommand(client.CHANGE_SCENARIO)

		if command != nil {
			if command.Type == "Manual" {
				aiYellow.HotswapPlanner(info.Yellow, manualMovementYellow)
				// aiBlue.HotswapPlanner(info.Blue, manualMovementBlue)
			} else if command.Type == "Game" {
				aiYellow.HotswapPlanner(info.Yellow, slowBrainYellow)
				aiBlue.HotswapPlanner(info.Blue, slowBrainBlue)
			}
		}

		if !gameInfo.HasField() || !gameInfo.State.IsValid() {
			continue
		}

		actionsYellow := aiYellow.GetActions(gameInfo)
		actionsBlue := aiBlue.GetActions(gameInfo)

		client.BroadcastActions(actionsYellow)
		client.BroadcastActions(actionsBlue)

		if config.IsSimulated() {
			simClientYellow.SendActions(actionsYellow)
			simClientBlue.SendActions(actionsBlue)

			handleSimulatedBall(gameInfo, simController, goalAreaTeleporter)
		} else {
			basestationClient.SendActions(actionsYellow)
			basestationClient.SendActions(actionsBlue)
		}
	}
}
