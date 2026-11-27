package action

// Init asks a real robot to (re)initialise. The simulator ignores it.
type Init struct {
	Id int
}

func (i *Init) RobotID() int { return i.Id }
