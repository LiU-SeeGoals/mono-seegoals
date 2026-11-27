package action

// Stop halts a robot and switches off its dribbler and kicker.
type Stop struct {
	Id int
}

func (s *Stop) RobotID() int { return s.Id }
