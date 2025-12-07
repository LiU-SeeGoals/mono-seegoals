package action

import (
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
	// Decides if the robot should dribble while moving
	Dribble bool
	// We need to know ID AND team to know how to update the pos
	Team info.Team

	KickSpeed int
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
	id := uint32(mv.Id)

	const distKp = 0.0000005
	const angleKp = 1.0
	mv.Dest.Angle = convAngle(mv.Dest.Angle)
	mv.Pos.Angle = convAngle(mv.Pos.Angle)

	// Angular velocity counter-clockwise [rad/s]
	// dx := mv.Pos.X - mv.Dest.X
	// dy := mv.Pos.Y - mv.Dest.Y
	dx := mv.Dest.X - mv.Pos.X
	dy := mv.Dest.Y - mv.Pos.Y
	angleDiff := mv.Dest.Angle - mv.Pos.Angle

	distance := math.Sqrt(dx*dx + dy*dy)
	maxSpeed := float64(0.5)

	speedCtrl := math.Max(math.Min(maxSpeed, distKp*distance), 0.002)

	maxAngleSpeed := 3.0
	angleCtrl := float32(math.Min(maxAngleSpeed, angleKp*float64(angleDiff)))

	// Invert robot angle to move in global coordinate
	// I.e. Rotation matrix around the z axis

	forward := float32(speedCtrl * (dx*math.Cos(-mv.Pos.Angle) - dy*math.Sin(-mv.Pos.Angle)))
	left := float32(speedCtrl * (dx*math.Sin(-mv.Pos.Angle) + dy*math.Cos(-mv.Pos.Angle)))

	dribblerSpeed := float32(0)
	if mv.Dribble {
		dribblerSpeed = 100 // in rpm, adjust as needed
	}

	localVel := &simulation.MoveLocalVelocity{
		Forward: &forward,
		Left:    &left,
		Angular: &angleCtrl,
	}

	// Create the move command and assign the local velocity to the oneof field
	moveCommand := &simulation.RobotMoveCommand{
		Command: &simulation.RobotMoveCommand_LocalVelocity{
			LocalVelocity: localVel,
		},
	}
	if mv.KickSpeed != 0 {
		kickSpeed := float32(mv.KickSpeed)
		return &simulation.RobotCommand{
			Id:            &id,
			MoveCommand:   moveCommand,
			DribblerSpeed: &dribblerSpeed,
			KickSpeed:     &kickSpeed,
		}
	}

	// Create the robot command with the move command
	return &simulation.RobotCommand{
		Id:            &id,
		MoveCommand:   moveCommand,
		DribblerSpeed: &dribblerSpeed,
	}
}

func (mv *MoveTo) TranslateSim() *simulation.RobotCommand {
	return mv.simulateRealMovement()

	id := uint32(mv.Id)

	// Angular velocity counter-clockwise [rad/s]
	dx := float64(mv.Pos.X - mv.Dest.X)
	dy := float64(mv.Pos.Y - mv.Dest.Y)
	angleDiff := mv.Dest.Angle - mv.Pos.Angle

	if angleDiff > math.Pi {
		angleDiff -= 2 * math.Pi
	}
	if angleDiff < -math.Pi {
		angleDiff += 2 * math.Pi
	}

	distance := math.Sqrt(dx*dx + dy*dy)
	maxSpeed := float64(0.5)
	DeAccDistance := float64(300) // The distance from target robot start to deaccelerate (measured in mm)
	speed := float32(math.Min(maxSpeed, (maxSpeed/DeAccDistance)*distance))

	maxAngleSpeed := float64(2)
	deAccAngleDistance := float64(0.5) // The distance from target robot start to deaccelerate (measured in rad)
	angle := float32(math.Min(maxAngleSpeed, (maxAngleSpeed/deAccAngleDistance)*float64(angleDiff)))

	// Compute the target direction in global space
	targetDirection := math.Atan2(-dy, -dx)
	targetDirection = math.Mod(targetDirection+math.Pi, 2*math.Pi) - math.Pi

	moveAngle := targetDirection - mv.Pos.Angle

	// Decompose movement into forward and leftward velocities
	forward := speed * float32(math.Cos(moveAngle)) // Forward velocity
	left := speed * float32(math.Sin(moveAngle))    // Leftward velocity

	dribblerSpeed := float32(0)
	if mv.Dribble {
		dribblerSpeed = 100 // in rpm, adjust as needed
	}

	kickSpeed := float32(mv.KickSpeed)

	localVel := &simulation.MoveLocalVelocity{
		Forward: &forward,
		Left:    &left,
		Angular: &angle,
	}

	// Create the move command and assign the local velocity to the oneof field
	moveCommand := &simulation.RobotMoveCommand{
		Command: &simulation.RobotMoveCommand_LocalVelocity{
			LocalVelocity: localVel,
		},
	}

	// Create the robot command with the move command
	return &simulation.RobotCommand{
		Id:            &id,
		MoveCommand:   moveCommand,
		DribblerSpeed: &dribblerSpeed,
		KickSpeed:     &kickSpeed,
	}
}

func (mt *MoveTo) TranslateReal() *robot_action.Command {
	// var dribble int32
	// if mt.Dribble {
	// 	dribble = 1
	// } else {
	// 	dribble = 0
	// }

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
	}
	return command_move
}

func (m *MoveTo) ToDTO() ActionDTO {
	return ActionDTO{
		Action:  robot_action.ActionType_MOVE_TO_ACTION,
		Id:      m.Id,
		PosX:    int32(m.Pos.X),
		PosY:    int32(m.Pos.Y),
		PosW:    float32(m.Pos.Angle),
		DestX:   int32(m.Dest.X),
		DestY:   int32(m.Dest.Y),
		DestW:   float32(m.Dest.Angle),
		Dribble: m.Dribble,
	}
}
