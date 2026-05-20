package action

import (
	"encoding/binary"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

const commandPosePayloadLen = 13

func commandByte(value int) []byte {
	if value == 0 {
		return nil
	}

	return []byte{byte(value)}
}

func commandPosePayload(flag int, pos info.Position, dest info.Position) []byte {
	payload := make([]byte, commandPosePayloadLen)
	payload[0] = byte(flag)

	putInt16(payload, 1, int32(pos.X+10000))
	putInt16(payload, 3, int32(pos.Y+10000))
	putInt16(payload, 5, int32(pos.Angle*1000))
	putInt16(payload, 7, int32(dest.X+10000))
	putInt16(payload, 9, int32(dest.Y+10000))
	putInt16(payload, 11, int32(dest.Angle*1000))

	return payload
}

func putInt16(payload []byte, offset int, value int32) {
	if value > 32767 {
		value = 32767
	} else if value < -32768 {
		value = -32768
	}

	binary.LittleEndian.PutUint16(payload[offset:], uint16(int16(value)))
}
