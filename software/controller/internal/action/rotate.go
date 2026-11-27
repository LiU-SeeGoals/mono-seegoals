package action

// Rotate spins in place; negative AngularVel is clockwise.
type Rotate struct {
	Id         int
	AngularVel int
}

func (r *Rotate) RobotID() int { return r.Id }
