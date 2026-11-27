package action

// Kick fires the kicker without moving.
type Kick struct {
	Id int
	// Simulator: m/s. Real robots: firmware kick code.
	KickSpeed int
}

func (k *Kick) RobotID() int { return k.Id }
