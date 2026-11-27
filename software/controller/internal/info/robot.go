package info

import (
	"container/list"
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/logger"
)

type Team int8
type ID uint8
type RobotTeam [TEAM_SIZE]*Robot

const (
	UNKNOWN Team = 0
	Yellow  Team = 1
	Blue    Team = 2
)

const (
	// Center2DribblerDist is the distance from robot center to the center of the
	// dribbler/front contact line [mm]. Tune this from real/sim measurements.
	Center2DribblerDist = 75.0
	BallRadius          = 21.5
	DribblerHalfWidth   = 35.0
	KickCenterTolerance = 15.0
)

func (t Team) String() string {
	switch t {
	case Yellow:
		return "Yellow"
	case Blue:
		return "Blue"
	default:
		return "UNKNOWN"
	}
}

type Robot struct {
	rawRobot
}

func NewRobot(id ID, team Team, history_capasity int) *Robot {
	return &Robot{
		rawRobot: rawRobot{
			active:          false,
			id:              id,
			team:            team,
			history:         list.New(),
			historyCapacity: history_capasity,
		},
	}
}

func (r *Robot) At(pos Position, threshold float64) bool {
	robotPos, err := r.GetPosition()
	if err != nil {
		logger.Logger.Errorf("Position retrieval failed - Robot: %v\n", err)
		return false
	}

	return robotPos.Distance(pos) < threshold
}

func (r *Robot) DribblerPos() Position {
	robotPos, _ := r.GetPosition()
	robotPos.X += Center2DribblerDist * math.Cos(robotPos.Angle)
	robotPos.Y += Center2DribblerDist * math.Sin(robotPos.Angle)
	return robotPos
}

func (r *Robot) HeldBallPos() Position {
	robotPos, _ := r.GetPosition()
	reach := Center2DribblerDist + BallRadius
	return Position{
		X: robotPos.X + reach*math.Cos(robotPos.Angle),
		Y: robotPos.Y + reach*math.Sin(robotPos.Angle),
	}
}

func BallLocalOffset(robotPos Position, ballPos Position) (forward float64, lateral float64) {
	dx := ballPos.X - robotPos.X
	dy := ballPos.Y - robotPos.Y

	forward = dx*math.Cos(robotPos.Angle) + dy*math.Sin(robotPos.Angle)
	lateral = -dx*math.Sin(robotPos.Angle) + dy*math.Cos(robotPos.Angle)
	return forward, lateral
}

func (r *Robot) BallLocalOffset(ballPos Position) (forward float64, lateral float64, ok bool) {
	robotPos, err := r.GetPosition()
	if err != nil {
		return 0, 0, false
	}
	forward, lateral = BallLocalOffset(robotPos, ballPos)
	return forward, lateral, true
}

func (r *Robot) Facing(target Position, threshold float64) bool {
	pos, err := r.GetPosition()
	if err != nil {
		return false
	}
	return pos.FacingPosition(target, threshold)
}

// GetVelocity averages the velocity between the newest sample and each older
// one in the history. X, Y are in mm/ms (= m/s); Angle is the angular
// velocity in rad/ms, computed from wrapped heading differences so crossing
// ±π does not produce a 2π spike.
func (r *Robot) GetVelocity() Position {
	if r.history.Len() < 2 {
		return Position{}
	}

	newest := r.history.Front().Value.(*rawRobotPos)
	var sum Position
	count := 0
	for e := r.history.Front().Next(); e != nil; e = e.Next() {
		older := e.Value.(*rawRobotPos)
		dt := float64(newest.time - older.time)
		if dt == 0 {
			continue
		}
		sum.X += (newest.pos.X - older.pos.X) / dt
		sum.Y += (newest.pos.Y - older.pos.Y) / dt
		sum.Z += (newest.pos.Z - older.pos.Z) / dt
		sum.Angle += NormalizeAngleDelta(newest.pos.Angle, older.pos.Angle) / dt
		count++
	}
	if count == 0 {
		return Position{}
	}
	n := float64(count)
	return Position{X: sum.X / n, Y: sum.Y / n, Z: sum.Z / n, Angle: sum.Angle / n}
}

// GetAcceleration is the mean rate of change of planar speed over the
// history, in mm/ms² (multiply by 1000 for m/s²). Positive means speeding up.
func (r *Robot) GetAcceleration() float64 {
	if r.history.Len() < 3 {
		return 0
	}

	sum := 0.0
	count := 0
	// The history is newest first: a is newer than b, b newer than c.
	for a, b, c := r.history.Front(), r.history.Front().Next(), r.history.Front().Next().Next(); c != nil; a, b, c = a.Next(), b.Next(), c.Next() {
		pa := a.Value.(*rawRobotPos)
		pb := b.Value.(*rawRobotPos)
		pc := c.Value.(*rawRobotPos)

		dtNew := float64(pa.time - pb.time)
		dtOld := float64(pb.time - pc.time)
		if dtNew <= 0 || dtOld <= 0 {
			continue
		}
		speedNew := pa.pos.Dist2d(pb.pos) / dtNew
		speedOld := pb.pos.Dist2d(pc.pos) / dtOld
		// Speeds are means over each segment, so they sit at segment midpoints.
		sum += (speedNew - speedOld) / ((dtNew + dtOld) / 2)
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func (r *Robot) GetTeam() Team {
	return r.team
}

func (r *Robot) String() string {

	pos, err := r.GetPosition()

	if err != nil {
		return ""
	}

	vel := r.GetVelocity()

	posString := fmt.Sprintf("(%f, %f, %f)", pos.X, pos.Y, pos.Angle)
	velString := fmt.Sprintf("(%f, %f, %f)", vel.X, vel.Y, vel.Angle)

	return fmt.Sprintf("id: %d, pos: %s, vel: %s", r.id, posString, velString)
}

func (r *Robot) ToDTO() RobotDTO {
	if !r.active {
		return RobotDTO{}
	}
	pos, _ := r.GetPosition()
	vel := r.GetVelocity()

	return RobotDTO{
		Id:       r.id,
		Team:     r.team,
		X:        pos.X,
		Y:        pos.Y,
		Angle:    pos.Angle,
		VelX:     vel.X,
		VelY:     vel.Y,
		VelAngle: vel.Angle,
	}
}

type RobotDTO struct {
	Id       ID
	Team     Team
	X        float64
	Y        float64
	Angle    float64
	VelX     float64
	VelY     float64
	VelAngle float64
}
