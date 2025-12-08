package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type KickAtPosition struct {
	GenericComposition
	targetPosition info.Position
	retrievingBall bool
}

func (k *KickAtPosition) String() string {
	return fmt.Sprintf("(Robot %d, KickAtPosition(%v))", k.id, k.targetPosition)
}

func NewKickAtPosition(team info.Team, id info.ID, targetPosition info.Position) *KickAtPosition {
	return &KickAtPosition{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		targetPosition: targetPosition,
		retrievingBall: true,
	}
}

func (kp *KickAtPosition) GetAction(gi *info.GameInfo) action.Action {
	const accuracy = 0.1 // Magic number warning

	robot := gi.State.GetRobot(kp.id, kp.team)
	ball := gi.State.GetBall()

	ballPos, _ := ball.GetEstimatedPosition()
	unitVector := kp.targetPosition.Sub(&ballPos).Normalize()
	unitVector.Angle = 0
	// We can now get a position that is behind the ball, opposite of the targetposition
	possessor := ball.GetPossessor()
	inPossession := possessor == robot

	kp.retrievingBall = !inPossession

	facingBall := robot.Facing(ballPos, accuracy*2)
	facingTarget := robot.Facing(kp.targetPosition, accuracy*2)

	fmt.Printf("[KickAtPosition] robot %d: inPossession=%v, retrievingBall=%v, facingBall=%v, facingTarget=%v\n",
		kp.id, inPossession, kp.retrievingBall, facingBall, facingTarget)
	if kp.retrievingBall && facingBall && facingTarget {
		fmt.Printf("[KickAtPosition] robot %d retrieving: aligned with ball and target -> MoveToBall\n", kp.id)
		return NewMoveToBall(kp.team, kp.id).GetAction(gi)
	} else if kp.retrievingBall {
		ballMargin := unitVector.Scale(350) // MagicNumber (100mm behind ball)
		lineUpPos := ballPos.Sub(&ballMargin)

		currPos, _ := robot.GetPosition()
		targetAngle := lineUpPos.AngleToPosition(kp.targetPosition)
		lineUpPos.Angle = currPos.Angle + info.NormalizeAngleDelta(targetAngle, currPos.Angle)

		move := NewMoveToPosition(kp.team, kp.id, lineUpPos)
		move.AvoidBall(true)
		moveAction := move.GetMoveToAction(gi)
		fmt.Printf("[KickAtPosition] robot %d retrieving: lining up at (%.1f, %.1f) angle %.1f° (facingBall=%v facingTarget=%v)\n",
			kp.id, lineUpPos.X, lineUpPos.Y, radToDeg(lineUpPos.Angle), facingBall, facingTarget)
		return &moveAction
	}

	// Robot is in possesion of the ball, but is not facing the target
	if inPossession && !robot.Facing(kp.targetPosition, accuracy) {
		// Since we cannot move and rotate with the ball reliably,
		// we need to move around the ball, while also rotating towards the ball at the same time
		// until we are lined up with the target, aswell as the ball.

		// We assume possesion means that the ball in proximity to the dribbler
		robotPos, _ := robot.GetPosition()

		angleDelta := info.NormalizeAngleDelta(robotPos.Angle, ballPos.AngleToPosition(kp.targetPosition))

		direction := math.Copysign(1.0, angleDelta)
		// Positive direction: Turn left
		// Negative direction: Turn right
		stepSize := 150.0 // millimeters
		radius := 115.0   // WARNING MAGIC NUMBER

		vecBallToRobot := robotPos.Sub(&ballPos)
		angleFromBall := vecBallToRobot.Angle

		tangentAngle := angleFromBall + (direction * math.Pi / 2)

		tangentPoint := robotPos.OnRadius(stepSize, tangentAngle)
		vecBallToNextPos := tangentPoint.Sub(&ballPos).Normalize().Scale(radius)
		nextPosition := ballPos.Add(&vecBallToNextPos)
		nextPosition.Angle = nextPosition.AngleToPosition(ballPos)

		fmt.Printf("Rotating around ball by stepping to robot %d (%.1f, %.1f) nextangle %.1f° (angleDelta=%.2f°)\n",
			kp.id, nextPosition.X, nextPosition.Y, radToDeg(nextPosition.Angle), radToDeg(angleDelta))
		// Create move action to the current target
		moveAction := action.MoveTo{
			Id:      int(kp.id),
			Team:    kp.team,
			Dest:    nextPosition,
			Dribble: false,
		}
		moveAction.Dest.Angle = nextPosition.Angle
		return &moveAction

	}

	// If we get here, it means we are in possession of the ball and that the robot is facing the target.
	// So we move forward and shoot
	runUpDistance := unitVector.Scale(100)
	destination := ballPos.Add(&runUpDistance)
	moveAction := action.MoveTo{
		Id:   int(kp.id),
		Dest: destination,
		Team: kp.team,
	}
	KickSpeed := float32(5)
	moveAction.KickSpeed = int(KickSpeed)
	fmt.Printf("[KickAtPosition] robot %d shooting: run-up target (%.1f, %.1f) angle %.1f° kickSpeed=%d\n",
		kp.id, destination.X, destination.Y, radToDeg(destination.Angle), moveAction.KickSpeed)
	return &moveAction
}

func (kp *KickAtPosition) Achieved(gi *info.GameInfo) bool {
	if kp.retrievingBall {
		return false
	}

	robot := gi.State.GetRobot(kp.id, kp.team)
	return gi.State.LostBall(robot)
}

func (kp *KickAtPosition) GetID() info.ID {
	return kp.id
}

// Helper
func radToDeg(rad float64) float64 {
	return rad * 180 / math.Pi
}
