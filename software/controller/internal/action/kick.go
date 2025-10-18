package action

import (
	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"github.com/LiU-SeeGoals/proto_go/simulation"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

type Kick struct {
	Id int
	// 1 is slow, 10 is faster, limits unknown
	KickSpeed int
}

func (k *Kick) TranslateSim() *simulation.RobotCommand {
	id := uint32(k.Id)
	kickSpeed := float32(k.KickSpeed) // in m/s
	Logger.Debugf("Kicking with speed %f", kickSpeed)

	return &simulation.RobotCommand{
		Id:        &id,
		KickSpeed: &kickSpeed,
	}
}

func (k *Kick) TranslateReal() *robot_action.Command {
	command_move := &robot_action.Command{
		CommandId: robot_action.ActionType_KICK_ACTION,
		RobotId:   int32(k.Id),
		KickSpeed: int32(k.KickSpeed),
	}
	return command_move
}

func (k *Kick) ToDTO() ActionDTO {
	return ActionDTO{
		Action: robot_action.ActionType_KICK_ACTION,
		Id:     k.Id,
		PosW:   float32(k.KickSpeed),
	}
}
