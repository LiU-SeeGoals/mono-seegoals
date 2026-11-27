package client

import (
	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/proto_go/robot_action"
)

// ActionDTO is the GameViewer's JSON view of one robot action.
type ActionDTO struct {
	Id     int                     `json:"Id"`
	Team   int                     `json:"Team,omitempty"`
	Role   string                  `json:"Role,omitempty"`
	Action robot_action.ActionType `json:"Action"`
	// Current pose (mm, mm, rad).
	PosX int32   `json:"PosX"`
	PosY int32   `json:"PosY"`
	PosW float32 `json:"PosW"`
	// Destination pose (mm, mm, rad).
	DestX   int32   `json:"DestX"`
	DestY   int32   `json:"DestY"`
	DestW   float32 `json:"DestW"`
	Dribble bool    `json:"Dribble"`
	// Planned path waypoints in field coordinates (mm), MoveTo only.
	Path []WaypointDTO `json:"Path,omitempty"`
}

type WaypointDTO struct {
	X float64 `json:"X"`
	Y float64 `json:"Y"`
}

func actionDTO(a action.Action) ActionDTO {
	switch a := a.(type) {
	case *action.MoveTo:
		var path []WaypointDTO
		if len(a.Path) > 0 {
			path = make([]WaypointDTO, 0, len(a.Path))
			for _, p := range a.Path {
				path = append(path, WaypointDTO{X: p.X, Y: p.Y})
			}
		}
		return ActionDTO{
			Action: robot_action.ActionType_MOVE_TO_ACTION,
			Id:     a.Id,
			Team:   int(a.Team),
			PosX:   int32(a.Pos.X), PosY: int32(a.Pos.Y), PosW: float32(a.Pos.Angle),
			DestX: int32(a.Dest.X), DestY: int32(a.Dest.Y), DestW: float32(a.Dest.Angle),
			Dribble: a.Dribble,
			Path:    path,
		}
	case *action.Stop:
		return ActionDTO{Action: robot_action.ActionType_STOP_ACTION, Id: a.Id}
	case *action.Kick:
		return ActionDTO{Action: robot_action.ActionType_KICK_ACTION, Id: a.Id, PosW: float32(a.KickSpeed)}
	case *action.Init:
		return ActionDTO{Action: robot_action.ActionType_INIT_ACTION, Id: a.Id}
	case *action.Move:
		return ActionDTO{
			Action: robot_action.ActionType_MOVE_ACTION,
			Id:     a.Id,
			DestX:  int32(a.Direction.AtVec(0)),
			DestY:  int32(a.Direction.AtVec(1)),
		}
	case *action.Rotate:
		return ActionDTO{Action: robot_action.ActionType_ROTATE_ACTION, Id: a.Id, PosW: float32(a.AngularVel)}
	default:
		return ActionDTO{Id: a.RobotID()}
	}
}
