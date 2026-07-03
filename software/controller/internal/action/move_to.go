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
	// AllowBehindGoalLine lets the executor keep destinations behind either goal
	// line. This is intended for keeper-only goal-mouth positioning.
	AllowBehindGoalLine bool
	// AllowGoalArea permits the robot to enter either goal/penalty area between
	// the penalty stretch and goal line. This is intended for the goalie only.
	AllowGoalArea bool
	// MinLinearSpeed optionally raises the simulator's linear speed floor for
	// motions such as close ball orbits. Zero keeps the default floor.
	MinLinearSpeed float64
	// Decides if the robot should dribble while moving
	Dribble bool
	// We need to know ID AND team to know how to update the pos
	Team info.Team

	KickSpeed int
	// SimKickSpeed is the physical simulator kick speed in m/s. If unset, the
	// simulator falls back to KickSpeed for older call sites that use it as m/s.
	SimKickSpeed float32
	// KickAngle is used by the simulator in degrees. Zero is a flat kick.
	KickAngle float32
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
	simKickAngle     float32
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

	const (
		linearKp     = 0.001
		angularKp    = 14.0
		maxLinearMPS = 1.0
		maxOmega     = 3.0
	)

	ex := mv.Dest.X - mv.Pos.X
	ey := mv.Dest.Y - mv.Pos.Y
	headingError := info.NormalizeAngleDelta(mv.Dest.Angle, mv.Pos.Angle)
	vxWorld := linearKp * ex
	vyWorld := linearKp * ey

	speed := math.Hypot(vxWorld, vyWorld)
	minLinearMPS := math.Min(maxLinearMPS, mv.MinLinearSpeed)
	if speed > maxLinearMPS {
		scale := maxLinearMPS / speed
		vxWorld *= scale
		vyWorld *= scale
	} else if speed > 0 && speed < minLinearMPS {
		// Close ball-orbit corrections deliberately request a speed floor so
		// the proportional position controller does not crawl into alignment.
		scale := minLinearMPS / speed
		vxWorld *= scale
		vyWorld *= scale
	}

	cosW := math.Cos(mv.Pos.Angle)
	sinW := math.Sin(mv.Pos.Angle)
	mv.simForward = float32(vxWorld*cosW + vyWorld*sinW)
	mv.simLeft = float32(-vxWorld*sinW + vyWorld*cosW)
	mv.simAngular = float32(math.Max(-maxOmega, math.Min(maxOmega, angularKp*headingError)))

	if mv.Dribble {
		mv.simDribblerSpeed = 100
	} else {
		mv.simDribblerSpeed = 0
	}

	if mv.KickSpeed != 0 {
		mv.simKickSpeed = mv.SimKickSpeed
		if mv.simKickSpeed == 0 {
			mv.simKickSpeed = float32(mv.KickSpeed)
		}
		mv.simCmd.KickSpeed = &mv.simKickSpeed
		if mv.KickAngle != 0 {
			mv.simKickAngle = mv.KickAngle
			mv.simCmd.KickAngle = &mv.simKickAngle
		} else {
			mv.simCmd.KickAngle = nil
		}
	} else {
		mv.simCmd.KickSpeed = nil
		mv.simCmd.KickAngle = nil
	}

	return &mv.simCmd
}

func (mv *MoveTo) TranslateSim() *simulation.RobotCommand {
	return mv.simulateRealMovement()
}

func (mt *MoveTo) TranslateReal() *robot_action.Command {
	kickSpeedReal := mt.KickSpeed
	dribbleSpeedReal := 0

	if mt.Dribble {
		dribbleSpeedReal = 1
	}
	if kickSpeedReal != 0 && dribbleSpeedReal == 1 {
		fmt.Println("Cannot send dribble and kick at same time, sending only kick")
	}

	if kickSpeedReal != 0 {
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
