package action

import (
	"testing"
	"github.com/LiU-SeeGoals/proto_go/robot_action"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		cmd  *robot_action.Command
	}{
		{
			name: "Simple kick command",
			cmd: &robot_action.Command{
				CommandId: int32(robot_action.ActionType_KICK_ACTION),
				RobotId:   1,
				KickSpeed: 1000,
				Pos: &robot_action.Vector3D{
					X: 1500,
					Y: 2000,
					W: 45.5,
				},
				Dest: &robot_action.Vector3D{
					X: 2000,
					Y: 2500,
					W: 0,
				},
				Direction: &robot_action.Vector2D{
					X: 50,
					Y: 75,
				},
				AngularVel: 90,
			},
		},
		{
			name: "Move to command with negative coordinates",
			cmd: &robot_action.Command{
				CommandId: int32(robot_action.ActionType_MOVE_TO_ACTION),
				RobotId:   3,
				KickSpeed: 0,
				Pos: &robot_action.Vector3D{
					X: -1000,
					Y: -500,
					W: 180.0,
				},
				Dest: &robot_action.Vector3D{
					X: 1000,
					Y: 1500,
					W: 0,
				},
				Direction: &robot_action.Vector2D{
					X: -100,
					Y: -50,
				},
				AngularVel: -45,
			},
		},
		{
			name: "Stop command with zeros",
			cmd: &robot_action.Command{
				CommandId: int32(robot_action.ActionType_STOP_ACTION),
				RobotId:   5,
				KickSpeed: 0,
				Pos: &robot_action.Vector3D{
					X: 0,
					Y: 0,
					W: 0,
				},
				Dest: &robot_action.Vector3D{
					X: 0,
					Y: 0,
					W: 0,
				},
				Direction: &robot_action.Vector2D{
					X: 0,
					Y: 0,
				},
				AngularVel: 0,
			},
		},
		{
			name: "Max values",
			cmd: &robot_action.Command{
				CommandId: int32(robot_action.ActionType_ROTATE_ACTION),
				RobotId:   7,
				KickSpeed: 32767,
				Pos: &robot_action.Vector3D{
					X: 2147483647,
					Y: 2147483647,
					W: 359.9,
				},
				Dest: &robot_action.Vector3D{
					X: 2147483647,
					Y: 2147483647,
					W: 0,
				},
				Direction: &robot_action.Vector2D{
					X: 100,
					Y: 100,
				},
				AngularVel: 2147483647,
			},
		},
		{
			name: "Direction clamping test",
			cmd: &robot_action.Command{
				CommandId: int32(robot_action.ActionType_MOVE_ACTION),
				RobotId:   2,
				KickSpeed: 500,
				Pos: &robot_action.Vector3D{
					X: 500,
					Y: 500,
					W: 90.5,
				},
				Dest: &robot_action.Vector3D{
					X: 1000,
					Y: 1000,
					W: 0,
				},
				Direction: &robot_action.Vector2D{
					X: 1000, // Should be clamped to 100
					Y: -1000, // Should be clamped to -100
				},
				AngularVel: 180,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := EncodeCommand(tt.cmd)
			if err != nil {
				t.Fatalf("EncodeCommand failed: %v", err)
			}

			// Check size
			if len(encoded) != CommandEncodedSize {
				t.Errorf("Expected encoded size %d, got %d", CommandEncodedSize, len(encoded))
			}

			// Decode
			decoded, err := DecodeCommand(encoded)
			if err != nil {
				t.Fatalf("DecodeCommand failed: %v", err)
			}

			// Verify round-trip
			if decoded.CommandId != tt.cmd.CommandId {
				t.Errorf("CommandId mismatch: expected %d, got %d", tt.cmd.CommandId, decoded.CommandId)
			}

			if decoded.RobotId != tt.cmd.RobotId {
				t.Errorf("RobotId mismatch: expected %d, got %d", tt.cmd.RobotId, decoded.RobotId)
			}

			if decoded.KickSpeed != tt.cmd.KickSpeed {
				t.Errorf("KickSpeed mismatch: expected %d, got %d", tt.cmd.KickSpeed, decoded.KickSpeed)
			}

			if decoded.Pos.X != tt.cmd.Pos.X {
				t.Errorf("Pos.X mismatch: expected %d, got %d", tt.cmd.Pos.X, decoded.Pos.X)
			}

			if decoded.Pos.Y != tt.cmd.Pos.Y {
				t.Errorf("Pos.Y mismatch: expected %d, got %d", tt.cmd.Pos.Y, decoded.Pos.Y)
			}

			// W is stored as int16 tenths of degrees, so allow small precision loss
			if diff := decoded.Pos.W - tt.cmd.Pos.W; diff < -0.1 || diff > 0.1 {
				t.Errorf("Pos.W mismatch: expected %f, got %f (diff: %f)", tt.cmd.Pos.W, decoded.Pos.W, diff)
			}

			if decoded.Dest.X != tt.cmd.Dest.X {
				t.Errorf("Dest.X mismatch: expected %d, got %d", tt.cmd.Dest.X, decoded.Dest.X)
			}

			if decoded.Dest.Y != tt.cmd.Dest.Y {
				t.Errorf("Dest.Y mismatch: expected %d, got %d", tt.cmd.Dest.Y, decoded.Dest.Y)
			}

			// Direction gets clamped
			expectedDirX := tt.cmd.Direction.X
			if expectedDirX < -100 {
				expectedDirX = -100
			} else if expectedDirX > 100 {
				expectedDirX = 100
			}
			if decoded.Direction.X != expectedDirX {
				t.Errorf("Direction.X mismatch: expected %d, got %d (input was %d)", expectedDirX, decoded.Direction.X, tt.cmd.Direction.X)
			}

			expectedDirY := tt.cmd.Direction.Y
			if expectedDirY < -100 {
				expectedDirY = -100
			} else if expectedDirY > 100 {
				expectedDirY = 100
			}
			if decoded.Direction.Y != expectedDirY {
				t.Errorf("Direction.Y mismatch: expected %d, got %d (input was %d)", expectedDirY, decoded.Direction.Y, tt.cmd.Direction.Y)
			}

			if decoded.AngularVel != tt.cmd.AngularVel {
				t.Errorf("AngularVel mismatch: expected %d, got %d", tt.cmd.AngularVel, decoded.AngularVel)
			}
		})
	}
}

func TestEncodeInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *robot_action.Command
		wantErr bool
	}{
		{
			name:    "Nil command",
			cmd:     nil,
			wantErr: true,
		},
		{
			name: "Invalid action type",
			cmd: &robot_action.Command{
				CommandId: 10,
				RobotId:   1,
			},
			wantErr: true,
		},
		{
			name: "Invalid robot ID",
			cmd: &robot_action.Command{
				CommandId: 0,
				RobotId:   256,
			},
			wantErr: true,
		},
		{
			name: "Negative robot ID",
			cmd: &robot_action.Command{
				CommandId: 0,
				RobotId:   -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodeCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeCommand error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecodeInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		buf     []byte
		wantErr bool
	}{
		{
			name:    "Empty buffer",
			buf:     []byte{},
			wantErr: true,
		},
		{
			name:    "Too short",
			buf:     make([]byte, 29),
			wantErr: true,
		},
		{
			name:    "Too long",
			buf:     make([]byte, 31),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeCommand(tt.buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeCommand error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkEncodeCommand(b *testing.B) {
	cmd := &robot_action.Command{
		CommandId: int32(robot_action.ActionType_MOVE_TO_ACTION),
		RobotId:   1,
		KickSpeed: 500,
		Pos: &robot_action.Vector3D{
			X: 1500,
			Y: 2000,
			W: 45.5,
		},
		Dest: &robot_action.Vector3D{
			X: 2000,
			Y: 2500,
			W: 0,
		},
		Direction: &robot_action.Vector2D{
			X: 50,
			Y: 75,
		},
		AngularVel: 90,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncodeCommand(cmd)
	}
}

func BenchmarkDecodeCommand(b *testing.B) {
	cmd := &robot_action.Command{
		CommandId: int32(robot_action.ActionType_MOVE_TO_ACTION),
		RobotId:   1,
		KickSpeed: 500,
		Pos: &robot_action.Vector3D{
			X: 1500,
			Y: 2000,
			W: 45.5,
		},
		Dest: &robot_action.Vector3D{
			X: 2000,
			Y: 2500,
			W: 0,
		},
		Direction: &robot_action.Vector2D{
			X: 50,
			Y: 75,
		},
		AngularVel: 90,
	}

	buf, _ := EncodeCommand(cmd)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecodeCommand(buf)
	}
}
