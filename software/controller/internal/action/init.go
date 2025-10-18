package action

import (
	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"github.com/LiU-SeeGoals/proto_go/simulation"
)

type Init struct {
	Id int
}

// Do nothing, only implemented to satisfy interface
func (i *Init) TranslateSim() *simulation.RobotCommand {
	id := uint32(i.Id)
	return &simulation.RobotCommand{
		Id: &id,
	}
}

func (i *Init) TranslateReal() *robot_action.Command {

	command_move := &robot_action.Command{
		CommandId: robot_action.ActionType_INIT_ACTION,
		RobotId:   int32(i.Id),
	}
	return command_move
}

func (i *Init) ToDTO() ActionDTO {
	return ActionDTO{
		Action: robot_action.ActionType_INIT_ACTION,
		Id:     i.Id,
	}
}
