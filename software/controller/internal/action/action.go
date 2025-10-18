package action

import (
	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"github.com/LiU-SeeGoals/proto_go/simulation"
)

type Action interface {
	//----------------------------------------------------------------------------------------------
	// TranslateReal
	//----------------------------------------------------------------------------------------------
	TranslateReal() *robot_action.Command
	//------------------------------------------------------------------//
	// TranslateSim translates the action to simulation proto message	//
	// (there are a lot of wrapper proto messages :(                    //
	//------------------------------------------------------------------//
	TranslateSim() *simulation.RobotCommand
	//----------------------------------------------------------------------------------------------
	// ToDTO
	//----------------------------------------------------------------------------------------------
	ToDTO() ActionDTO
}

type ActionDTO struct {
	// The id of the robot.
	Id     int                     `json:"Id"`
	Action robot_action.ActionType `json:"Action"`
	// Current position of Robot, vector contains (x,y,w)
	PosX int32   `json:"PosX"`
	PosY int32   `json:"PosY"`
	PosW float32 `json:"PosW"`
	// Goal destination of Robot, vector contains (x,y,w)
	DestX int32   `json:"DestX"`
	DestY int32   `json:"DestY"`
	DestW float32 `json:"DestW"`
	// Decides if the robot should dribble while moving
	Dribble bool `json:"Dribble"`
}
