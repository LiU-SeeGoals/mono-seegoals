package info

import "time"

type TrackedRobot struct {
	Id        uint32
	Team      Team
	Pos       Position
	Vel       Position
	Orientation float64
	VelAngular float64
	Timestamp  float64
	Valid      bool
	updatedAt  int64
}

func NewTrackedRobot(id uint32, team Team) *TrackedRobot {
	return &TrackedRobot{
		Id:   id,
		Team: team,
	}
}

func (tr *TrackedRobot) SetTracked(pos Position, vel Position, orientation float64, velAngular float64, ts float64) {
	tr.Pos = pos
	tr.Vel = vel
	tr.Orientation = orientation
	tr.VelAngular = velAngular
	tr.Timestamp = ts
	tr.Valid = true
	tr.updatedAt = time.Now().UnixMilli()
}

func (tr *TrackedRobot) GetTrackedPosition() (Position, bool) {
	if !tr.Valid {
		return Position{}, false
	}
	return tr.Pos, true
}

func (tr *TrackedRobot) GetTrackedVelocity() (Position, bool) {
	if !tr.Valid {
		return Position{}, false
	}
	return tr.Vel, true
}


func (tr *TrackedRobot) GetFreshTrackedVelocity(now time.Time, maxAge time.Duration) (Position, bool) {
	if !tr.Valid || now.UnixMilli()-tr.updatedAt > maxAge.Milliseconds() {
		return Position{}, false
	}
	return tr.Vel, true
}
