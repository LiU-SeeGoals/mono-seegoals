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

    //------------------------------------------------------
    // Current pose
    //------------------------------------------------------

    curX := mv.Pos.X
    curY := mv.Pos.Y
    curW := mv.Pos.Angle

    //------------------------------------------------------
    // Goal pose
    //------------------------------------------------------

    destX := mv.Dest.X
    destY := mv.Dest.Y
    destW := mv.Dest.Angle

    //------------------------------------------------------
    // Position errors (world frame)
    //------------------------------------------------------

    ex := destX - curX
    ey := destY - curY

    ew := info.NormalizeAngleDelta(destW, curW)

    //------------------------------------------------------
    // Firmware gains
    //------------------------------------------------------

    const Kx = 0.001
    const Ky = 0.001
    const Kw = 14.0

    //------------------------------------------------------
    // World frame velocity
    //------------------------------------------------------

    vxWorld := Kx * ex
    vyWorld := Ky * ey

    //------------------------------------------------------
    // Optional velocity saturation
    //------------------------------------------------------

    const maxLinear = 100000000

    speed := math.Hypot(vxWorld, vyWorld)

    if speed > maxLinear {
        scale := maxLinear / speed
        vxWorld *= scale
        vyWorld *= scale
    }

    //------------------------------------------------------
    // World -> Robot frame
    //------------------------------------------------------

    cosW := math.Cos(curW)
    sinW := math.Sin(curW)

    forward := float32(
        vxWorld*cosW +
            vyWorld*sinW)

    left := float32(
        -vxWorld*sinW +
            vyWorld*cosW)

    //------------------------------------------------------
    // Heading controller
    //------------------------------------------------------

    omega := Kw * ew

    const maxOmega = 3.0

    omega = math.Max(
        -maxOmega,
        math.Min(maxOmega, omega),
    )

    //------------------------------------------------------
    // Send command
    //------------------------------------------------------

    mv.simForward = forward
    mv.simLeft = left
    mv.simAngular = float32(omega)

    if mv.Dribble {
        mv.simDribblerSpeed = 100
    } else {
        mv.simDribblerSpeed = 0
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
