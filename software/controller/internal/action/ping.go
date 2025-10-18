package action

import (
	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"github.com/LiU-SeeGoals/proto_go/simulation"
)

type Ping struct {
	Id int
}

// Do nothing, only implemented to satisfy interface
func (i *Ping) TranslateSim() *simulation.RobotCommand {
	id := uint32(i.Id)
	return &simulation.RobotCommand{
		Id: &id,
	}
}

func (s *Ping) TranslateReal() *robot_action.Command {
	command := &robot_action.Command{
		CommandId: robot_action.ActionType_PING,
		RobotId:   int32(s.Id),
	}
	return command
}

// Do nothing, only implemented to satisfy interface
func (s *Ping) ToDTO() ActionDTO {
	return ActionDTO{
		Id: s.Id,
	}
}
