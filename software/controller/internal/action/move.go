package action

import (
	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"github.com/LiU-SeeGoals/proto_go/simulation"
	"gonum.org/v1/gonum/mat"
)

// Forward is x=0, y=1, Backward is x=0, y=-1, Left is x=-1, y=0, Right is x=1, y=0
// the size of the vector sets the speed of the robot
type Move struct {
	Id        int
	Direction *mat.VecDense // 2D vector, first value is x, second is y
}

func (s *Move) TranslateSim() *simulation.RobotCommand {

	id := uint32(s.Id)
	angular := float32(0) // No angular velocity currently, adjust as needed
	forward := float32(s.Direction.AtVec(0))
	left := float32(s.Direction.AtVec(1))

	// Create the local velocity command
	localVel := &simulation.MoveLocalVelocity{
		Forward: &forward,
		Left:    &left,
		Angular: &angular,
	}

	// Create the move command and assign the local velocity to the oneof field
	moveCommand := &simulation.RobotMoveCommand{
		Command: &simulation.RobotMoveCommand_LocalVelocity{
			LocalVelocity: localVel,
		},
	}

	// Create the robot command with the move command
	return &simulation.RobotCommand{
		Id:          &id,
		MoveCommand: moveCommand,
	}
}

func (s *Move) TranslateReal() *robot_action.Command {
	command := &robot_action.Command{
		CommandId: robot_action.ActionType_MOVE_ACTION,
		RobotId:   int32(s.Id),
		Direction: &robot_action.Vector2D{
			X: int32(s.Direction.AtVec(0)),
			Y: int32(s.Direction.AtVec(1)),
		},
	}
	return command
}

func (s *Move) ToDTO() ActionDTO {
	return ActionDTO{
		Action: robot_action.ActionType_MOVE_ACTION,
		Id:     s.Id,
		DestX:  int32(s.Direction.AtVec(0)),
		DestY:  int32(s.Direction.AtVec(1)),
		DestW:  0,
	}
}
