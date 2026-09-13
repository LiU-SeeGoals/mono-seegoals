package action

import (
	"encoding/binary"
	"fmt"
	"github.com/LiU-SeeGoals/proto_go/robot_action"
)

// CommandEncoder handles custom binary encoding of robot commands
// Protocol: 32 bytes total (fills the NRF24 32-byte limit)
//
// Byte Layout:
// [0]     : Action Type (uint8) - enum 0-5
// [1]     : Robot ID (uint8) - 1-7
// [2-3]   : Kick Speed (int16) - mm/s
// [4-5]   : Pos X (int16) - mm
// [6-7]   : Pos Y (int16) - mm
// [8-9]   : Dest X (int16) - mm
// [10-11] : Dest Y (int16) - mm
// [12-13] : Direction X (int16) - raw value
// [14-15] : Direction Y (int16) - raw value
// [16-17] : Angular Vel (int16) - degrees/sec
// [18-19] : Orientation W (int16) - radians * 1000 (milliradians)
//
// Total: 20 bytes of fields, 12 bytes reserved (zero-filled)

const (
	CommandEncodedSize = 32

	offsetActionType   = 0
	offsetRobotID      = 1
	offsetKickSpeed    = 2
	offsetPosX         = 4
	offsetPosY         = 6
	offsetDestX        = 8
	offsetDestY        = 10
	offsetDirectionX   = 12
	offsetDirectionY   = 14
	offsetAngularVel   = 16
	offsetOrientationW = 18
)

// EncodeCommand converts a robot_action.Command to a 32-byte binary buffer
func EncodeCommand(cmd *robot_action.Command) ([]byte, error) {
	if cmd == nil {
		return nil, fmt.Errorf("command cannot be nil")
	}


	if cmd.CommandId < 0 || cmd.CommandId > 5 {
		return nil, fmt.Errorf("invalid action type: %d", cmd.CommandId)
	}

	if cmd.RobotId < 0 || cmd.RobotId > 255 {
		return nil, fmt.Errorf("invalid robot id: %d", cmd.RobotId)
	}

	var posX, posY, posW, destX, destY, dirX, dirY int16
	if cmd.Pos != nil {
		posX = int16(cmd.Pos.X)
		posY = int16(cmd.Pos.Y)
		posW = int16(cmd.Pos.W * 1000.0)
	}
	if cmd.Dest != nil {
		destX = int16(cmd.Dest.X)
		destY = int16(cmd.Dest.Y)
	}
	if cmd.Direction != nil {
		dirX = int16(cmd.Direction.X)
		dirY = int16(cmd.Direction.Y)
	}

	buf := make([]byte, CommandEncodedSize)

	buf[offsetActionType] = uint8(cmd.CommandId)
	buf[offsetRobotID] = uint8(cmd.RobotId)
	binary.LittleEndian.PutInt16(buf[offsetKickSpeed:], int16(cmd.KickSpeed))

	binary.LittleEndian.PutInt16(buf[offsetPosX:], posX)
	binary.LittleEndian.PutInt16(buf[offsetPosY:], posY)

	binary.LittleEndian.PutInt16(buf[offsetDestX:], destX)
	binary.LittleEndian.PutInt16(buf[offsetDestY:], destY)

	binary.LittleEndian.PutInt16(buf[offsetDirectionX:], dirX)
	binary.LittleEndian.PutInt16(buf[offsetDirectionY:], dirY)

	binary.LittleEndian.PutInt16(buf[offsetAngularVel:], int16(cmd.AngularVel))

	binary.LittleEndian.PutInt16(buf[offsetOrientationW:], posW)

	return buf, nil
}

// DecodeCommand converts a 32-byte binary buffer back to a robot_action.Command
func DecodeCommand(buf []byte) (*robot_action.Command, error) {
	if len(buf) != CommandEncodedSize {
		return nil, fmt.Errorf("invalid buffer size: expected %d, got %d", CommandEncodedSize, len(buf))
	}

	cmd := &robot_action.Command{
		CommandId:  int32(buf[offsetActionType]),
		RobotId:    int32(buf[offsetRobotID]),
		KickSpeed:  int32(binary.LittleEndian.Int16(buf[offsetKickSpeed:])),
		Pos: &robot_action.Vector3D{
			X: int32(binary.LittleEndian.Int16(buf[offsetPosX:])),
			Y: int32(binary.LittleEndian.Int16(buf[offsetPosY:])),
			W: float32(int16(binary.LittleEndian.Int16(buf[offsetOrientationW:]))) / 1000.0,
		},
		Dest: &robot_action.Vector3D{
			X: int32(binary.LittleEndian.Int16(buf[offsetDestX:])),
			Y: int32(binary.LittleEndian.Int16(buf[offsetDestY:])),
			W: 0,
		},
		Direction: &robot_action.Vector2D{
			X: int32(binary.LittleEndian.Int16(buf[offsetDirectionX:])),
			Y: int32(binary.LittleEndian.Int16(buf[offsetDirectionY:])),
		},
		AngularVel: int32(binary.LittleEndian.Int16(buf[offsetAngularVel:])),
	}

	return cmd, nil
}
