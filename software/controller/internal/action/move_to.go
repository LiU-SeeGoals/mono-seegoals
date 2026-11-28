package action

import "github.com/LiU-SeeGoals/controller/internal/info"

// MoveTo drives a robot to Dest. Real robots close the loop onboard from
// (Pos, Dest); in simulation client.SimMotion does it.
type MoveTo struct {
	Id   int
	Team info.Team
	// Pos is the robot's current pose (X, Y in mm, Angle in rad).
	Pos info.Position
	// Dest is the target pose.
	Dest info.Position
	// Path is the planned route, for GameViewer display only.
	Path []info.Position

	// AllowOutsideField keeps destinations in the boundary area instead of
	// clamping them to the touchlines. The goal-line guard stays active.
	AllowOutsideField bool
	// AllowBehindGoalLine permits destinations behind either goal line
	// (keeper goal-mouth positioning).
	AllowBehindGoalLine bool
	// AllowGoalArea permits entering either defense area (goalie only).
	AllowGoalArea bool

	Dribble bool

	// KickSpeed arms the kicker when non-zero. On real robots it is the
	// firmware's kick code; in simulation it is used as m/s unless
	// SimKickSpeed is set.
	KickSpeed int
	// SimKickSpeed is the simulator kick speed in m/s.
	SimKickSpeed float32
	// KickAngle is the simulator chip angle in degrees; zero is a flat kick.
	KickAngle float32
	// MinLinearSpeed raises the simulator controller's speed floor (m/s) for
	// motions such as close ball orbits. Zero keeps the default.
	MinLinearSpeed float64
	// MaxLinearSpeed caps the drive speed (m/s), e.g. 1.5 m/s rule limit
	// during STOP. Zero means no limit beyond the robot's own.
	MaxLinearSpeed float64
}

func (m *MoveTo) RobotID() int { return m.Id }
