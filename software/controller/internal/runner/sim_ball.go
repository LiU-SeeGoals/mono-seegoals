package runner

import (
	"math"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	ballPlacementTouchlineMarginMM = 200.0
	ballPlacementCornerMarginMM    = 200.0
	// Goal kicks are free kicks placed 1 m from the goal line.
	ballPlacementGoalKickDepthMM     = 1000.0
	ballPlacementTeleportToleranceMM = 10.0
	// SSL penalty marks are measured from the center of the goal being attacked.
	penaltyMarkGoalDistanceDivisionAMM = 8000.0
	penaltyMarkGoalDistanceDivisionBMM = 6000.0
	mmToM                              = 1.0 / 1000.0
)

// simulatedBallHandler stands in for a human ball placer in simulation: it
// teleports the ball for placements, kickoffs and penalties, and recentres a
// ball that has not been seen for 5 s.
type simulatedBallHandler struct {
	target   info.Position
	pending  bool
	since    time.Time
	lastSend time.Time
	// lastState is the referee state of the previous frame; kickoff and
	// penalty placement happen once, on entering the preparation.
	lastState    info.RefState
	lastStateSet bool
}

type ballTeleporter interface {
	TeleportBall(x, y float32)
}

func (h *simulatedBallHandler) handle(gameInfo *info.GameInfo, simController ballTeleporter) {

	ball := gameInfo.State.GetBall()
	ballPos, ballTime, _ := ball.GetPositionTime()

	ge := gameInfo.Status.GetGameEvent()
	currentState := ge.GetCurrentState()
	previousState := ge.GetPreviousState() // first frame: use the referee's edge
	if h.lastStateSet {
		previousState = h.lastState
	}
	h.lastState, h.lastStateSet = currentState, true
	if ge.IsPlacementStop() {
		target, needed := refereePlacementTarget(ge, ballPos)
		// Derive a fallback only when the referee has supplied no target.
		if ge.GetDesignatedPosition() == nil {
			target, needed = outsideFieldPlacement(gameInfo, ballPos)
		}
		if !needed {
			h.pending = false
			return
		}
		now := time.Now()
		if !h.pending || h.target.Dist2d(target) > 10 {
			h.target, h.pending, h.since = target, true, now
		}
		if now.Sub(h.since) < 300*time.Millisecond || now.Sub(h.lastSend) < 250*time.Millisecond {
			return
		}
		// Let STOP positioning clear the destination before inserting the ball.
		for _, team := range []info.Team{info.Blue, info.Yellow} {
			for _, robot := range gameInfo.State.GetTeam(team) {
				if robot == nil {
					continue
				}
				pos, err := robot.GetPosition()
				if err == nil && pos.Dist2d(target) < 600 {
					return
				}
			}
		}
		teleportBallMillimeters(simController, target)
		h.lastSend = now
		return
	}
	h.pending = false
	switch currentState {
	case info.STATE_KICKOFF_PREPARATION:
		if previousState == info.STATE_HALTED || previousState == info.STATE_STOPPED {
			teleportBallMillimeters(simController, info.Position{})
			return
		}
	case info.STATE_PENALTY_PREPARATION:
		if previousState == info.STATE_HALTED || previousState == info.STATE_STOPPED {
			if mark, ok := penaltyMarkPosition(gameInfo, ge.GetTeamWithPossession()); ok {
				teleportBallMillimeters(simController, mark)
				return
			}
		}
	}
	if time.Now().UnixMilli()-ballTime > 5000 {
		teleportBallMillimeters(simController, info.Position{})
	}
}

func refereePlacementTarget(gameEvent *info.GameEvent, ballPos info.Position) (info.Position, bool) {
	if gameEvent == nil {
		return info.Position{}, false
	}
	state := gameEvent.GetCurrentState()
	if state != info.STATE_STOPPED && state != info.STATE_BALL_PLACEMENT {
		return info.Position{}, false
	}

	designatedPosition := gameEvent.GetDesignatedPosition()
	if designatedPosition == nil || designatedPosition.Len() < 2 {
		return info.Position{}, false
	}

	target := info.Position{
		X: designatedPosition.AtVec(0),
		Y: designatedPosition.AtVec(1),
	}
	return target, ballPos.Dist2d(target) > ballPlacementTeleportToleranceMM
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

func penaltyMarkPosition(gameInfo *info.GameInfo, attackingTeam info.Team) (info.Position, bool) {
	if gameInfo == nil || !gameInfo.HasField() {
		return info.Position{}, false
	}

	field := gameInfo.FieldSize()
	if field.X <= 0 {
		return info.Position{}, false
	}

	ownGoalSign := gameInfo.OwnHalfXSign(attackingTeam)
	opponentGoalX := -ownGoalSign * field.X / 2
	goalDistance := penaltyMarkGoalDistanceDivisionBMM
	if gameInfo.Status.GetDivision() == info.DivisionA {
		goalDistance = penaltyMarkGoalDistanceDivisionAMM
	}
	return info.Position{
		X: opponentGoalX + ownGoalSign*goalDistance,
		Y: 0,
	}, true
}

func teleportBallMillimeters(simController ballTeleporter, pos info.Position) {
	simController.TeleportBall(float32(pos.X*mmToM), float32(pos.Y*mmToM))
}
