package client

import (
	"encoding/binary"
	"fmt"

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
//	[22-31] zero
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
)

// baseStationCommand is one decoded wire frame. Values keep the integer and
// float32 precision of the original protobuf-based path so frames are
// byte-identical to what the firmware has always received.
type baseStationCommand struct {
	kind         robot_action.ActionType
	robotID      int32
	kickSpeed    int32
	posX, posY   int32
	posW         float32 // rad
	destX, destY int32
	destW        float32 // rad
	dirX, dirY   int32
	aux          int32
}

// baseStationCommandFor translates an action. ok is false for actions the
// real robots do not support.
func baseStationCommandFor(a action.Action) (baseStationCommand, bool) {
	switch a := a.(type) {
	case *action.Stop:
		return baseStationCommand{kind: robot_action.ActionType_STOP_ACTION, robotID: int32(a.Id)}, true
	case *action.Kick:
		return baseStationCommand{kind: robot_action.ActionType_KICK_ACTION, robotID: int32(a.Id), kickSpeed: int32(a.KickSpeed)}, true
	case *action.Init:
		return baseStationCommand{kind: robot_action.ActionType_INIT_ACTION, robotID: int32(a.Id)}, true
	case *action.MoveTo:
		cmd := baseStationCommand{
			kind:    robot_action.ActionType_MOVE_TO_ACTION,
			robotID: int32(a.Id),
			posX:    int32(a.Pos.X), posY: int32(a.Pos.Y), posW: float32(a.Pos.Angle),
			destX: int32(a.Dest.X), destY: int32(a.Dest.Y), destW: float32(a.Dest.Angle),
		}
		if a.KickSpeed != 0 {
			// The firmware's kick action keeps position control while it
			// charges and always runs the dribbler, so aux stays zero.
			cmd.kind = robot_action.ActionType_KICK_ACTION
			cmd.kickSpeed = int32(a.KickSpeed)
		} else if a.Dribble {
			cmd.aux = 1
		}
		return cmd, true
	case *action.Move:
		return baseStationCommand{
			kind:    robot_action.ActionType_MOVE_ACTION,
			robotID: int32(a.Id),
			dirX:    int32(a.Direction.AtVec(0)),
			dirY:    int32(a.Direction.AtVec(1)),
		}, true
	case *action.Rotate:
		return baseStationCommand{kind: robot_action.ActionType_ROTATE_ACTION, robotID: int32(a.Id), aux: int32(a.AngularVel)}, true
	case *action.MoveRemote:
		cmd := baseStationCommand{
			kind:    robot_action.ActionType_MOVE_ACTION,
			robotID: int32(a.Id),
			posW:    float32(a.Dest.Z),
			// The remote-control firmware path expects X/Y swapped.
			dirX: int32(a.Direction.AtVec(1)), dirY: int32(a.Direction.AtVec(0)),
			destX: int32(a.Dest.Y), destY: int32(a.Dest.X), destW: float32(a.Dest.Z),
			kickSpeed: int32(a.Speed),
		}
		if a.Dribble {
			cmd.aux = 1
		}
		return cmd, true
	default:
		return baseStationCommand{}, false
	}
}

func (c baseStationCommand) isMotion() bool {
	switch c.kind {
	case robot_action.ActionType_MOVE_TO_ACTION, robot_action.ActionType_MOVE_ACTION, robot_action.ActionType_ROTATE_ACTION:
		return true
	default:
		return false
	}
}

func (c baseStationCommand) encode() ([]byte, error) {
	if c.kind < 0 || c.kind > 5 {
		return nil, fmt.Errorf("invalid action type: %d", c.kind)
	}
	if c.robotID < 0 || c.robotID > 255 {
		return nil, fmt.Errorf("invalid robot id: %d", c.robotID)
	}
	buf := make([]byte, CommandEncodedSize)
	put := func(offset int, v int16) { binary.LittleEndian.PutUint16(buf[offset:], uint16(v)) }

	buf[offsetActionType] = uint8(c.kind)
	buf[offsetRobotID] = uint8(c.robotID)
	put(offsetKickSpeed, int16(c.kickSpeed))
	put(offsetPosX, int16(c.posX))
	put(offsetPosY, int16(c.posY))
	put(offsetDestX, int16(c.destX))
	put(offsetDestY, int16(c.destY))
	put(offsetDirectionX, int16(c.dirX))
	put(offsetDirectionY, int16(c.dirY))
	put(offsetAux, int16(c.aux))
	put(offsetPosW, int16(c.posW*1000.0))
	put(offsetDestW, int16(c.destW*1000.0))
	return buf, nil
}

func decodeBaseStationCommand(buf []byte) (baseStationCommand, error) {
	if len(buf) != CommandEncodedSize {
		return baseStationCommand{}, fmt.Errorf("invalid buffer size: expected %d, got %d", CommandEncodedSize, len(buf))
	}
	get := func(offset int) int16 { return int16(binary.LittleEndian.Uint16(buf[offset:])) }
	return baseStationCommand{
		kind:      robot_action.ActionType(buf[offsetActionType]),
		robotID:   int32(buf[offsetRobotID]),
		kickSpeed: int32(get(offsetKickSpeed)),
		posX:      int32(get(offsetPosX)),
		posY:      int32(get(offsetPosY)),
		posW:      float32(get(offsetPosW)) / 1000.0,
		destX:     int32(get(offsetDestX)),
		destY:     int32(get(offsetDestY)),
		destW:     float32(get(offsetDestW)) / 1000.0,
		dirX:      int32(get(offsetDirectionX)),
		dirY:      int32(get(offsetDirectionY)),
		aux:       int32(get(offsetAux)),
	}, nil
}
