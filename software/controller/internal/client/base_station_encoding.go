package client

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/proto_go/robot_action"
)

// Robot command wire format: 32 bytes (one nRF24 payload), little endian,
// decoded by firmware/shared/src/robot_command.c.
//
//	[0]     action type (robot_action.ActionType)
//	[1]     robot ID
//	[2-3]   kick speed / kick code (int16)
//	[4-7]   current X, Y (int16, mm)
//	[8-11]  destination X, Y (int16, mm)
//	[12-15] direction X, Y (int16, raw)
//	[16-17] aux (int16): dribbler on/off for MOVE_TO and MOVE, angular
//	        velocity for ROTATE
//	[18-19] current heading (int16, mrad)
//	[20-21] destination heading (int16, mrad)
//	[22-23] max linear speed for MOVE_TO (int16, mm/s; 0 = no limit).
//	        Firmware that predates this field ignores it.
//	[24-31] zero
const (
	CommandEncodedSize = 32

	offsetActionType = 0
	offsetRobotID    = 1
	offsetKickSpeed  = 2
	offsetPosX       = 4
	offsetPosY       = 6
	offsetDestX      = 8
	offsetDestY      = 10
	offsetDirectionX = 12
	offsetDirectionY = 14
	offsetAux        = 16
	offsetPosW       = 18
	offsetDestW      = 20
	offsetMaxSpeed   = 22
)

// BaseStationCommand is one decoded wire frame. Values keep the integer and
// float32 precision of the original protobuf-based path so frames are
// byte-identical to what the firmware has always received.
type BaseStationCommand struct {
	Kind         robot_action.ActionType
	RobotID      int32
	KickSpeed    int32
	PosX, PosY   int32
	PosW         float32 // rad
	DestX, DestY int32
	DestW        float32 // rad
	DirX, DirY   int32
	Aux          int32
	MaxSpeed     int32 // mm/s, 0 = no limit
}

// BaseStationCommandFor translates an action. ok is false for actions the
// real robots do not support.
func BaseStationCommandFor(a action.Action) (BaseStationCommand, bool) {
	switch a := a.(type) {
	case *action.Stop:
		return BaseStationCommand{Kind: robot_action.ActionType_STOP_ACTION, RobotID: int32(a.Id)}, true
	case *action.Kick:
		return BaseStationCommand{Kind: robot_action.ActionType_KICK_ACTION, RobotID: int32(a.Id), KickSpeed: int32(a.KickSpeed)}, true
	case *action.Init:
		return BaseStationCommand{Kind: robot_action.ActionType_INIT_ACTION, RobotID: int32(a.Id)}, true
	case *action.MoveTo:
		cmd := BaseStationCommand{
			Kind:    robot_action.ActionType_MOVE_TO_ACTION,
			RobotID: int32(a.Id),
			PosX:    int32(a.Pos.X), PosY: int32(a.Pos.Y), PosW: float32(a.Pos.Angle),
			DestX: int32(a.Dest.X), DestY: int32(a.Dest.Y), DestW: float32(a.Dest.Angle),
		}
		if a.MaxLinearSpeed > 0 {
			cmd.MaxSpeed = int32(math.Min(a.MaxLinearSpeed*1000, math.MaxInt16))
		}
		if a.KickSpeed != 0 {
			// The firmware's kick action keeps position control while it
			// charges and always runs the dribbler, so Aux stays zero.
			cmd.Kind = robot_action.ActionType_KICK_ACTION
			cmd.KickSpeed = int32(a.KickSpeed)
		} else if a.Dribble {
			cmd.Aux = 1
		}
		return cmd, true
	case *action.Move:
		return BaseStationCommand{
			Kind:    robot_action.ActionType_MOVE_ACTION,
			RobotID: int32(a.Id),
			DirX:    int32(a.Direction.AtVec(0)),
			DirY:    int32(a.Direction.AtVec(1)),
		}, true
	case *action.Rotate:
		return BaseStationCommand{Kind: robot_action.ActionType_ROTATE_ACTION, RobotID: int32(a.Id), Aux: int32(a.AngularVel)}, true
	case *action.MoveRemote:
		cmd := BaseStationCommand{
			Kind:    robot_action.ActionType_MOVE_ACTION,
			RobotID: int32(a.Id),
			PosW:    float32(a.Dest.Z),
			// The remote-control firmware path expects X/Y swapped.
			DirX: int32(a.Direction.AtVec(1)), DirY: int32(a.Direction.AtVec(0)),
			DestX: int32(a.Dest.Y), DestY: int32(a.Dest.X), DestW: float32(a.Dest.Z),
			KickSpeed: int32(a.Speed),
		}
		if a.Dribble {
			cmd.Aux = 1
		}
		return cmd, true
	default:
		return BaseStationCommand{}, false
	}
}

func (c BaseStationCommand) isMotion() bool {
	switch c.Kind {
	case robot_action.ActionType_MOVE_TO_ACTION, robot_action.ActionType_MOVE_ACTION, robot_action.ActionType_ROTATE_ACTION:
		return true
	default:
		return false
	}
}

func (c BaseStationCommand) Encode() ([]byte, error) {
	if c.Kind < 0 || c.Kind > 5 {
		return nil, fmt.Errorf("invalid action type: %d", c.Kind)
	}
	if c.RobotID < 0 || c.RobotID > 255 {
		return nil, fmt.Errorf("invalid robot id: %d", c.RobotID)
	}
	buf := make([]byte, CommandEncodedSize)
	put := func(offset int, v int16) { binary.LittleEndian.PutUint16(buf[offset:], uint16(v)) }

	buf[offsetActionType] = uint8(c.Kind)
	buf[offsetRobotID] = uint8(c.RobotID)
	put(offsetKickSpeed, int16(c.KickSpeed))
	put(offsetPosX, int16(c.PosX))
	put(offsetPosY, int16(c.PosY))
	put(offsetDestX, int16(c.DestX))
	put(offsetDestY, int16(c.DestY))
	put(offsetDirectionX, int16(c.DirX))
	put(offsetDirectionY, int16(c.DirY))
	put(offsetAux, int16(c.Aux))
	put(offsetPosW, int16(c.PosW*1000.0))
	put(offsetDestW, int16(c.DestW*1000.0))
	put(offsetMaxSpeed, int16(c.MaxSpeed))
	return buf, nil
}

func DecodeBaseStationCommand(buf []byte) (BaseStationCommand, error) {
	if len(buf) != CommandEncodedSize {
		return BaseStationCommand{}, fmt.Errorf("invalid buffer size: expected %d, got %d", CommandEncodedSize, len(buf))
	}
	get := func(offset int) int16 { return int16(binary.LittleEndian.Uint16(buf[offset:])) }
	return BaseStationCommand{
		Kind:      robot_action.ActionType(buf[offsetActionType]),
		RobotID:   int32(buf[offsetRobotID]),
		KickSpeed: int32(get(offsetKickSpeed)),
		PosX:      int32(get(offsetPosX)),
		PosY:      int32(get(offsetPosY)),
		PosW:      float32(get(offsetPosW)) / 1000.0,
		DestX:     int32(get(offsetDestX)),
		DestY:     int32(get(offsetDestY)),
		DestW:     float32(get(offsetDestW)) / 1000.0,
		DirX:      int32(get(offsetDirectionX)),
		DirY:      int32(get(offsetDirectionY)),
		Aux:       int32(get(offsetAux)),
		MaxSpeed:  int32(get(offsetMaxSpeed)),
	}, nil
}
