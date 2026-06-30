package demos

import (
	"fmt"
	"math"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	plan "github.com/LiU-SeeGoals/controller/internal/ai/plan"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/simulator"
)

const (
	ballPlacementTouchlineMarginMM = 200.0
	ballPlacementCornerMarginMM    = 200.0
	ballPlacementGoalKickDepthMM   = 0.0
	mmToM                          = 1.0 / 1000.0
)

func handleSimulatedBall(gameInfo *info.GameInfo, simController *simulator.SimControl) {

	ball := gameInfo.State.GetBall()
	ballPos, ballTime, _ := ball.GetPositionTime()
	if placement, ok := outsideFieldPlacement(gameInfo, ballPos); ok {
		teleportBallMillimeters(simController, placement)
	} else if time.Now().UnixMilli()-ballTime > 5000 {
		teleportBallMillimeters(simController, info.Position{Y: 0})
	}

	ge := gameInfo.Status.GetGameEvent()
	previousState := ge.GetPreviousState()
	currentState := ge.GetCurrentState()
	switch currentState {
	case info.STATE_KICKOFF_PREPARATION:
		if previousState == info.STATE_HALTED || previousState == info.STATE_STOPPED {
			// fmt.Println("teleported ball (new kickoff)")
			teleportBallMillimeters(simController, info.Position{Y: 0})
		}
	case info.STATE_FREE_KICK:
	case info.STATE_HALTED, info.STATE_STOPPED:
	case info.STATE_PENALTY_PREPARATION, info.STATE_TIMEOUT:
	case info.STATE_PLAYING:
	case info.STATE_BALL_PLACEMENT:
		if previousState != info.STATE_BALL_PLACEMENT {
			toX := ge.GetDesignatedPosition().At(0, 0)
			toY := ge.GetDesignatedPosition().At(1, 0)
			fmt.Printf("teleported ball (%f, %f) (ball placement %s)\n", toX, toY, ge.GetTeamWithPossession())
			teleportBallMillimeters(simController, info.Position{X: toX, Y: toY})
		}
	default:
	}
}

func teleportBallMillimeters(simController *simulator.SimControl, pos info.Position) {
	simController.TeleportBall(float32(pos.X*mmToM), float32(pos.Y*mmToM))
}

func outsideFieldPlacement(gameInfo *info.GameInfo, ballPos info.Position) (info.Position, bool) {
	if gameInfo == nil || !gameInfo.HasField() {
		return info.Position{}, false
	}

	field := gameInfo.FieldSize()
	halfLength := field.X / 2
	halfWidth := field.Y / 2
	if halfLength <= 0 || halfWidth <= 0 {
		return info.Position{}, false
	}

	overX := math.Abs(ballPos.X) - halfLength
	overY := math.Abs(ballPos.Y) - halfWidth
	if overX <= 0 && overY <= 0 {
		return info.Position{}, false
	}

	xSign := signOrOne(ballPos.X)
	ySign := signOrOne(ballPos.Y)
	if overY >= overX {
		return info.Position{
			X: clampFloat(ballPos.X, -halfLength+ballPlacementTouchlineMarginMM, halfLength-ballPlacementTouchlineMarginMM),
			Y: ySign * (halfWidth - ballPlacementTouchlineMarginMM),
		}, true
	}

	xDepth := ballPlacementGoalKickDepthMM
	if ballLeftNearCorner(ballPos, halfWidth) || goalLineExitWasCornerKick(gameInfo, xSign) {
		xDepth = ballPlacementCornerMarginMM
	}

	return info.Position{
		X: xSign * (halfLength - xDepth),
		Y: ySign * (halfWidth - ballPlacementCornerMarginMM),
	}, true
}

func goalLineExitWasCornerKick(gameInfo *info.GameInfo, goalLineSign float64) bool {
	if gameInfo == nil || gameInfo.State == nil || !gameInfo.State.KickedBall.Valid {
		return false
	}

	return gameInfo.State.KickedBall.RobotTeam == defendingTeamAtGoalLine(gameInfo, goalLineSign)
}

func defendingTeamAtGoalLine(gameInfo *info.GameInfo, goalLineSign float64) info.Team {
	blueDefendsPositiveX := gameInfo.Status.GetBlueTeamOnPositiveHalf()
	if goalLineSign > 0 {
		if blueDefendsPositiveX {
			return info.Blue
		}
		return info.Yellow
	}
	if blueDefendsPositiveX {
		return info.Yellow
	}
	return info.Blue
}

func ballLeftNearCorner(ballPos info.Position, halfWidth float64) bool {
	return math.Abs(ballPos.Y) >= halfWidth
}

func signOrOne(value float64) float64 {
	if value < 0 {
		return -1
	}
	return 1
}

func clampFloat(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
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
		teleportBallMillimeters(simController, info.Position{Y: 0})
		simController.SetPresentRobots(teamYellow, teamBlue)

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

			handleSimulatedBall(gameInfo, simController)
		} else {
			basestationClient.SendActions(actionsYellow)
			basestationClient.SendActions(actionsBlue)
		}
	}
}
