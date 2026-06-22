package action

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/proto_go/robot_action"
	"github.com/LiU-SeeGoals/proto_go/simulation"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

type MoveTo struct {
	// The id of the robot.
	Id int
	// Current position of Robot, vector contains (x,y,w)
	Pos info.Position
	// Goal destination of Robot, vector contains (x,y,w)
	Dest info.Position
	// Planned path (e.g. RRT/RRT*) from (roughly) current position to final goal.
	// Used for visualization in the GameViewer.
	Path []info.Position
	// AllowOutsideField lets the executor keep destinations in the SSL boundary
	// area instead of clamping them to the playable touchlines. The goal-line
	// safety guard remains active.
	AllowOutsideField bool
	// Decides if the robot should dribble while moving
	Dribble bool
	// We need to know ID AND team to know how to update the pos
	Team info.Team

	KickSpeed int
	// Pre-allocated protobuf objects to avoid repeated heap allocations
	simCmd       simulation.RobotCommand
	simMoveCmd   simulation.RobotMoveCommand
	simLocalVel  simulation.MoveLocalVelocity
	simAllocated bool
	// Pre-allocated scalar fields (protobuf needs pointers)
	simId            uint32
	simForward       float32
	simLeft          float32
	simAngular       float32
	simDribblerSpeed float32
	simKickSpeed     float32
}

func convAngle(angle float64) float64 {

	if angle > math.Pi {
		return angle - 2*math.Pi
	} else if angle < -math.Pi {
		return angle + 2*math.Pi
	}

	return angle
}

func (mv *MoveTo) simulateRealMovement() *simulation.RobotCommand {
	// Initialize pre-allocated structs on first call
	if !mv.simAllocated {
		mv.simId = uint32(mv.Id)
		mv.simLocalVel = simulation.MoveLocalVelocity{
			Forward: &mv.simForward,
			Left:    &mv.simLeft,
			Angular: &mv.simAngular,
		}
		mv.simMoveCmd = simulation.RobotMoveCommand{
			Command: &simulation.RobotMoveCommand_LocalVelocity{
				LocalVelocity: &mv.simLocalVel,
			},
		}
		mv.simCmd = simulation.RobotCommand{
			Id:            &mv.simId,
			MoveCommand:   &mv.simMoveCmd,
			DribblerSpeed: &mv.simDribblerSpeed,
		}
		mv.simAllocated = true
	}

	const maxLinearSpeed = 0.65
	const slowdownDistance = 1000.0
	const minLinearSpeed = 0.1
	const angleKp = 1.0
	mv.Dest.Angle = convAngle(mv.Dest.Angle)
	mv.Pos.Angle = convAngle(mv.Pos.Angle)

	dx := mv.Dest.X - mv.Pos.X
	dy := mv.Dest.Y - mv.Pos.Y
	angleDiff := info.NormalizeAngleDelta(mv.Dest.Angle, mv.Pos.Angle)
	distance := math.Sqrt(dx*dx + dy*dy)

	speed := 0.0
	if distance > 1 {
		speed = math.Min(maxLinearSpeed, maxLinearSpeed/slowdownDistance*distance)
		speed = math.Max(speed, minLinearSpeed)
	}

	maxAngleSpeed := 3.0
	angleCtrl := float32(math.Max(-maxAngleSpeed, math.Min(maxAngleSpeed, angleKp*float64(angleDiff))))

	forward := float32(0)
	left := float32(0)
	if distance > 1 {
		unitX := dx / distance
		unitY := dy / distance
		forward = float32(speed * (unitX*math.Cos(-mv.Pos.Angle) - unitY*math.Sin(-mv.Pos.Angle)))
		left = float32(speed * (unitX*math.Sin(-mv.Pos.Angle) + unitY*math.Cos(-mv.Pos.Angle)))
	}

	// Update pre-allocated scalar values in place (pointers already wired up)
	mv.simForward = forward
	mv.simLeft = left
	mv.simAngular = angleCtrl

	if mv.Dribble {
		mv.simDribblerSpeed = 100
	} else {
		mv.simDribblerSpeed = 0
	}

	if mv.KickSpeed != 0 {
		mv.simKickSpeed = float32(mv.KickSpeed)
		mv.simCmd.KickSpeed = &mv.simKickSpeed
	} else {
		mv.simCmd.KickSpeed = nil
	}

	return &mv.simCmd
}

func (mv *MoveTo) TranslateSim() *simulation.RobotCommand {
	return mv.simulateRealMovement()
}

func (mt *MoveTo) TranslateReal() *robot_action.Command {
	// Robots only take binary commands for kick and dribblespeed.
	// Either 0 or 1.
	kickSpeedReal := 0

	if mt.KickSpeed != 0 {
		kickSpeedReal = 1

	}
	dribbleSpeedReal := 0

	if mt.Dribble {
		dribbleSpeedReal = 1
	}
	if kickSpeedReal == 1 && dribbleSpeedReal == 1 {
		fmt.Println("Cannot send dribble and kick at same time, sending only kick")
	}

	if kickSpeedReal == 1 {
		// fmt.Println("Kicking")
		dribbleSpeedReal = 1
		command_kick := &robot_action.Command{
			CommandId: robot_action.ActionType_KICK_ACTION,
			RobotId:   int32(mt.Id),
			Pos: &robot_action.Vector3D{
				X: int32(mt.Pos.X + 10000),
				Y: int32(mt.Pos.Y + 10000),
				W: float32(mt.Pos.Angle * 1000),
			},
			Dest: &robot_action.Vector3D{
				X: int32(mt.Dest.X + 10000),
				Y: int32(mt.Dest.Y + 10000),
				W: float32(mt.Dest.Angle * 1000),
			},
			KickSpeed: int32(kickSpeedReal),
		}
		return command_kick
	}

	command_move := &robot_action.Command{
		CommandId: robot_action.ActionType_MOVE_TO_ACTION,
		RobotId:   int32(mt.Id),
		Pos: &robot_action.Vector3D{
			X: int32(mt.Pos.X + 10000),
			Y: int32(mt.Pos.Y + 10000),
			W: float32(mt.Pos.Angle * 1000),
		},
		Dest: &robot_action.Vector3D{
			X: int32(mt.Dest.X + 10000),
			Y: int32(mt.Dest.Y + 10000),
			W: float32(mt.Dest.Angle * 1000),
		},
		AngularVel: int32(dribbleSpeedReal),
	}

	return command_move
}

func (m *MoveTo) ToDTO() ActionDTO {
	var pathDTO []WaypointDTO
	if len(m.Path) > 0 {
		pathDTO = make([]WaypointDTO, 0, len(m.Path))
		for _, p := range m.Path {
			pathDTO = append(pathDTO, WaypointDTO{X: p.X, Y: p.Y})
		}
	}

	return ActionDTO{
		Action:  robot_action.ActionType_MOVE_TO_ACTION,
		Id:      m.Id,
		Team:    int(m.Team),
		PosX:    int32(m.Pos.X),
		PosY:    int32(m.Pos.Y),
		PosW:    float32(m.Pos.Angle),
		DestX:   int32(m.Dest.X),
		DestY:   int32(m.Dest.Y),
		DestW:   float32(m.Dest.Angle),
		Dribble: m.Dribble,
		Path:    pathDTO,
	}
}
