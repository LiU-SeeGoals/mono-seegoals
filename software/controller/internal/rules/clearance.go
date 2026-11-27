// Package rules is the controller's single model of what the SSL rules allow
// right now: MatchRules answers "may this team touch the ball / how far must
// it stay away / how fast may it drive", and this file is the one table of
// distances every layer (referee formations, path planner, executor safety)
// uses to stay legal.
//
// All distances are millimetres, measured from the robot centre unless the
// name says otherwise.
package rules

// Geometry.
const (
	// RobotRadius is the SSL robot radius (rules: max 180 mm diameter). The
	// ball radius is info.BallRadius.
	RobotRadius = 90.0
)

// Distances required by the SSL rules (body or centre as noted). Our margins
// are added below; do not use these directly for motion targets.
const (
	// ruleStopBallDistance: during STOP every robot keeps 0.5 m from the ball.
	ruleStopBallDistance = 500.0
	// ruleRestartBallDistance: during a kickoff or free kick the defending
	// team keeps 0.5 m from the ball.
	ruleRestartBallDistance = 500.0
	// ruleStopSpeed: maximum robot speed during STOP and ball placement (m/s).
	ruleStopSpeed = 1.5
)

// Ball keep-out radii (robot centre to ball centre).
const (
	// StopBallKeepout is the STOP / ball placement keep-out: the rule
	// distance plus 200 mm for vision noise and braking.
	StopBallKeepout = ruleStopBallDistance + 200.0
	// RestartBallKeepout is the keep-out for the defending team during a
	// kickoff or free kick.
	RestartBallKeepout = ruleRestartBallDistance
	// FreeKickWallBallClearance is the distance a free-kick wall robot keeps
	// from the ball: just outside RestartBallKeepout.
	FreeKickWallBallClearance = RestartBallKeepout + 10.0
	// FreeKickPrepareDistance is where our kicker waits behind the ball while
	// the restart is still in STOP: outside StopBallKeepout.
	FreeKickPrepareDistance = StopBallKeepout + 50.0
)

// Defense-area clearances (robot centre to the area's edge).
const (
	// DefenseAreaMotionClearance inflates the defense areas when the path
	// planner treats them as obstacles.
	DefenseAreaMotionClearance = 100.0
	// DefenseAreaStoppedBodyClearance is how far the robot body stays from
	// either defense area while the ball is out of play.
	DefenseAreaStoppedBodyClearance = 300.0
	// DefenseAreaStoppedClearance is DefenseAreaStoppedBodyClearance as a
	// centre distance; the executor enforces it whenever the ball is out of
	// play and referee formations place robots outside it.
	DefenseAreaStoppedClearance = DefenseAreaStoppedBodyClearance + RobotRadius
	// SupportDefenseAreaClearance keeps support positions off the opponent's
	// defense area during open play.
	SupportDefenseAreaClearance = 200.0
)

// Goal-line clearance (robot centre to the goal line). The executor grows it
// with the robot's outward speed so the robot can brake in time.
const (
	GoalLineBaseClearance = 150.0
	GoalLineMaxClearance  = 1200.0
	// GoalLineReactionTime is the command-to-motion delay of the real robots (s).
	GoalLineReactionTime = 0.10
	// GoalLineBrakeDeceleration is the assumed braking deceleration (m/s²).
	GoalLineBrakeDeceleration = 1.50
)

// Restart formations.
const (
	// KickoffCenterCircleClearance keeps robots outside the centre circle
	// (radius 500 mm) during a kickoff, with 200 mm margin.
	KickoffCenterCircleClearance = 700.0
	// FreeKickFieldClearance keeps free-kick formation slots inside the field lines.
	FreeKickFieldClearance = 200.0
)
