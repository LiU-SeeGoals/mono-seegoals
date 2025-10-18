package action

import (
	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"github.com/LiU-SeeGoals/proto_go/simulation"
)

// Negative value rotates robot clockwise
type Rotate struct {
	Id         int
	AngularVel int
}

func (r *Rotate) TranslateSim() *simulation.RobotCommand {
	id := uint32(r.Id)
	angular := float32(r.AngularVel) // No angular velocity currently, adjust as needed
	forward := float32(0)
	left := float32(0)

	localVel := &simulation.MoveLocalVelocity{
		Forward: &forward,
		Left:    &left,
		Angular: &angular,
	}

	moveCommand := &simulation.RobotMoveCommand{
		Command: &simulation.RobotMoveCommand_LocalVelocity{
			LocalVelocity: localVel,
		},
	}

	return &simulation.RobotCommand{
		Id:          &id,
		MoveCommand: moveCommand,
	}
}

func (r *Rotate) TranslateReal() *robot_action.Command {
	command_move := &robot_action.Command{
		CommandId:  robot_action.ActionType_ROTATE_ACTION,
		RobotId:    int32(r.Id),
		AngularVel: int32(r.AngularVel),
	}
	return command_move
}


func (r *Rotate) ToDTO() ActionDTO {
	return ActionDTO{
		Action: robot_action.ActionType_ROTATE_ACTION,
		Id:     r.Id,
		PosW:   float32(r.AngularVel),
	}
}
