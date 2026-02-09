package info

type TrackedRobot struct {
	Id        uint32
	Team      Team
	Pos       Position
	Vel       Position
	Orientation float64
	VelAngular float64
	Timestamp  float64
	Valid      bool
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

