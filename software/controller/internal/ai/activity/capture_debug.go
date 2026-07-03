package ai

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/logger"
)

const (
	captureDebugEnabled           = true
	captureDebugPrintPeriod       = 250 * time.Millisecond
	captureDebugHeadingAnomaly    = math.Pi / 2
	captureDebugHeadingAnomalyDeg = 90.0
)

type captureDebugState struct {
	lastLog            time.Time
	robotHeading       float64
	destinationHeading float64
	largeCommand       bool
	initialized        bool
}

var (
	captureDebugMu     sync.Mutex
	captureDebugStates = map[string]captureDebugState{}
)

func printCaptureDebug(
	label string,
	team info.Team,
	id info.ID,
	robot *info.Robot,
	robotPos info.Position,
	ballPos info.Position,
	target info.Position,
	dest info.Position,
	headingErr float64,
	captureReady bool,
	ballCentered bool,
	dribble bool,
	kickSpeed int,
	margin float64,
) {
	if !captureDebugEnabled {
		return
	}

	const radToDeg = 180 / math.Pi
	robotToBallHeading := robotPos.AngleToPosition(ballPos)
	kickTargetHeading := ballPos.AngleToPosition(target)
	commandHeadingError := info.NormalizeAngleDelta(dest.Angle, robotPos.Angle)
	largeHeadingCommand := math.Abs(commandHeadingError) >= captureDebugHeadingAnomaly

	key := fmt.Sprintf("%s:%s:%d", label, team, id)
	now := time.Now()
	captureDebugMu.Lock()
	state := captureDebugStates[key]
	robotHeadingJump := 0.0
	destinationHeadingJump := 0.0
	if state.initialized {
		robotHeadingJump = info.NormalizeAngleDelta(robotPos.Angle, state.robotHeading)
		destinationHeadingJump = info.NormalizeAngleDelta(dest.Angle, state.destinationHeading)
	}
	headingAnomaly := math.Abs(robotHeadingJump) >= captureDebugHeadingAnomaly ||
		math.Abs(destinationHeadingJump) >= captureDebugHeadingAnomaly ||
		(largeHeadingCommand && !state.largeCommand)
	periodicLogDue := state.lastLog.IsZero() || now.Sub(state.lastLog) >= captureDebugPrintPeriod
	state.robotHeading = robotPos.Angle
	state.destinationHeading = dest.Angle
	state.largeCommand = largeHeadingCommand
	state.initialized = true
	if !periodicLogDue && !headingAnomaly {
		captureDebugStates[key] = state
		captureDebugMu.Unlock()
		return
	}
	state.lastLog = now
	captureDebugStates[key] = state
	captureDebugMu.Unlock()

	along, side, lineOK := lineErrorToTarget(robotPos, ballPos, target)
	forward := math.NaN()
	lateral := math.NaN()
	if robot != nil {
		if f, l, ok := robot.BallLocalOffset(ballPos); ok {
			forward = f
			lateral = l
		}
	}
	dribblerDist := math.NaN()
	if robot != nil {
		driblerpos := robot.DribblerPos()
		dribblerDist = driblerpos.Dist2d(ballPos)
	}

	// Use structured fields so real-match logs can distinguish an AI target
	// jump from a vision heading jump or a slow readiness gate. In particular,
	// command_heading_error_deg will be close to +/-180 when the AI asks the
	// real robot to perform the unexpected half-turn seen during kick setup.
	logger.Logger.Infow(
		"capture-debug",
		"phase", label,
		"team", team.String(),
		"robot_id", int(id),
		"line_ok", lineOK,
		"along_mm", along,
		"side_mm", side,
		"ball_local_forward_mm", forward,
		"ball_local_lateral_mm", lateral,
		"dribbler_distance_mm", dribblerDist,
		"heading_error_deg", headingErr*radToDeg,
		"capture_ready", captureReady,
		"ball_centered", ballCentered,
		"dribble", dribble,
		"kick_speed", kickSpeed,
		"margin_mm", margin,
		"robot_x_mm", robotPos.X,
		"robot_y_mm", robotPos.Y,
		"robot_heading_deg", robotPos.Angle*radToDeg,
		"robot_to_ball_heading_deg", robotToBallHeading*radToDeg,
		"ball_x_mm", ballPos.X,
		"ball_y_mm", ballPos.Y,
		"target_x_mm", target.X,
		"target_y_mm", target.Y,
		"kick_target_heading_deg", kickTargetHeading*radToDeg,
		"destination_x_mm", dest.X,
		"destination_y_mm", dest.Y,
		"destination_heading_deg", dest.Angle*radToDeg,
		"command_heading_error_deg", commandHeadingError*radToDeg,
		"robot_heading_jump_deg", robotHeadingJump*radToDeg,
		"destination_heading_jump_deg", destinationHeadingJump*radToDeg,
		"large_heading_command", largeHeadingCommand,
		"heading_anomaly_threshold_deg", captureDebugHeadingAnomalyDeg,
	)
}
