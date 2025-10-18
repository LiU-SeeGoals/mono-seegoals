package info

import (
	"container/list"
	"fmt"
	"math"

	. "github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/controller/internal/tracker"
)

type Team int8
type ID uint8
type RobotTeam [TEAM_SIZE]*Robot

const (
	UNKNOWN Team = 0
	Yellow  Team = 1
	Blue    Team = 2
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
	tracker tracker.RobotTracker
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
		Logger.Errorf("Position retrieval failed - Robot: %v\n", err)
		return false
	}

	return robotPos.Distance(pos) < threshold
}


func (r *Robot) DribblerPos() Position {

	robotPos, _ := r.GetPosition()
	robotPos.X += 90 * math.Cos(robotPos.Angle) // WARN: Magic number
	robotPos.Y += 90 * math.Sin(robotPos.Angle) // WARN: Magic number
	return robotPos
}


func (r *Robot) Facing(target Position, threshold float64) bool {
	pos, err := r.GetPosition()
	if err != nil {
		return false
	}
	return pos.FacingPosition(target, threshold) 
}

func (r *Robot) GetVelocity() Position {
	if r.history.Len() < 2 {
		return Position{0, 0, 0, 0}
	}

	element := r.history.Front()
	robot := element.Value.(*rawRobotPos)

	sum_deltas := Position{}

	for e := r.history.Front().Next(); e != nil; e = e.Next() {
		robot2 := e.Value.(*rawRobotPos)
		dPos := robot2.pos.Sub(&robot.pos)
		dt := float64(robot2.time - robot.time)
		// TODO: lets add exponential decay so that the most recent deltas have more weight
		scaled := dPos.Scale(1 / dt)
		sum_deltas = sum_deltas.Add(&scaled)
	}
	return sum_deltas.Scale(1 / float64(r.history.Len()-1))
}

func (r *Robot) GetAcceleration() float64 {
	if r.history.Len() < 3 {
		return float64(0) // Not enough data points to calculate acceleration
	}

	accelerations := float64(0)
	for f, s, t := r.history.Front(), r.history.Front().Next(), r.history.Front().Next().Next(); t != nil; f, s, t = f.Next(), s.Next(), t.Next() {

		robot1 := f.Value.(*rawRobotPos)
		robot2 := s.Value.(*rawRobotPos)
		robot3 := t.Value.(*rawRobotPos)

		vel1 := robot2.pos.Sub(&robot1.pos)
		vel2 := robot3.pos.Sub(&robot2.pos)

		dist1 := vel1.Norm()
		dist2 := vel2.Norm()

		dt := float64(robot3.time - robot1.time)

		accelerations += (dist2 - dist1) / dt

	}

	return accelerations / float64(r.history.Len()-1)
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
