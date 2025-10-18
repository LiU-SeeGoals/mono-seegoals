package ai

import (
	"fmt"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

type MoveWithBallToPosition struct {
	GenericComposition
	targetPosition info.Position
	retrievingBall bool
}

func (m *MoveWithBallToPosition) String() string {
	return fmt.Sprintf("Robot %d, MoveWithBallToPosition(%v)", m.id, m.targetPosition)
}

func NewMoveWithBallToPosition(team info.Team, id info.ID, dest info.Position) *MoveWithBallToPosition {
	return &MoveWithBallToPosition{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		targetPosition: dest,
		retrievingBall: false,
	}
}

// Moves to a target position with ball in the direction of the target.
// When robot is at the target position, it rotates to face the target angle.
// If the robot is not the possessor of the ball, it moves to the ball.
func (fb *MoveWithBallToPosition) GetAction(gi *info.GameInfo) action.Action {

	robot := gi.State.GetRobot(fb.id, fb.team)
	robotPosition, err := robot.GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Robot: %v\n", err)
		return NewStop(fb.id).GetAction(gi)
	}
	if !robot.IsActive() {
		return NewStop(fb.id).GetAction(gi)
	}
	if !fb.retrievingBall { // Check if it lost the ball
		fb.retrievingBall = gi.State.LostBall(robot)
	}

	move := NewMoveToBall(fb.team, fb.id)
	if fb.retrievingBall && move.Achieved(gi) { // We have achivied in retrieving the ball
		Logger.Debug("MoveWithBallToPosition: Ball retrieved")
		fb.retrievingBall = false

	} else if fb.retrievingBall { // We are still working on getting the ball
		Logger.Debug("MoveWithBallToPosition: Retrieving ball")
		return move.GetAction(gi)

	}

	Logger.Debug("MoveWithBallToPosition: Moving with ball to position")

	// Move to target and keep the ball facing the direction in which
	// we are moving to avoid dropping it.
	targetDistance := robotPosition.Distance(fb.targetPosition)
	if targetDistance > 100 { // WARN: Magic number
		moveAction := NewMoveToPosition(fb.team, fb.id, fb.targetPosition).GetMoveToAction(gi) // Make a copy of target so we can change the angle

		angleToTarget := robotPosition.AngleToPosition(moveAction.Dest)
		moveAction.Dest.Angle = angleToTarget

		act := action.MoveTo{
			Id:   int(fb.id),
			Team: fb.team,
			Pos:  robotPosition,
			Dest: moveAction.Dest,

			Dribble: true,
		}
		// return &act
		return &act
	}

	// We are at target position, now rotate to target angle
	act := action.MoveTo{
		Id:   int(fb.id),
		Team: fb.team,
		Pos:  robotPosition,
		Dest: fb.targetPosition,

		Dribble: true,
	}
	return &act

}

func (m *MoveWithBallToPosition) Achieved(gi *info.GameInfo) bool {
	ballPosition, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Ball: %v\n", err)
		return false
	}
	robotPosition, err := gi.State.GetRobot(m.id, m.team).GetPosition()
	if err != nil {
		Logger.Errorf("Position retrieval failed - Robot: %v\n", err)
		return false
	}

	distanceLeft := ballPosition.Distance(m.targetPosition)
	const distance_threshold = 100
	const angle_threshold = 0.1
	distance_achieved := distanceLeft <= distance_threshold

	angle_diff := robotPosition.AngleDistance(m.targetPosition)
	angle_achieved := angle_diff <= angle_threshold
	return distance_achieved && angle_achieved
}

func (m *MoveWithBallToPosition) GetID() info.ID {
	return m.id
}
