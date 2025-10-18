package action

import (
	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"github.com/LiU-SeeGoals/proto_go/simulation"
)

type Stop struct {
	Id int
}

func (s *Stop) TranslateSim() *simulation.RobotCommand {
	id := uint32(s.Id)
	angular := float32(0)
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

func (s *Stop) TranslateReal() *robot_action.Command {
	command_move := &robot_action.Command{
		CommandId: robot_action.ActionType_STOP_ACTION,
		RobotId:   int32(s.Id),
	}
	return command_move
}

func (s *Stop) ToDTO() ActionDTO {
	return ActionDTO{
		Action: robot_action.ActionType_STOP_ACTION,
		Id:     s.Id,
	}
}
