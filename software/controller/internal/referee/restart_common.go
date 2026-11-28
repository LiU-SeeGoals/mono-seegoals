package referee

import (
	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"math"
	"sort"
	"time"
)

const (
	// Restart kicks hand control to the normal planner as soon as the ball is
	// in play (info.BallInPlayDistance). Keep the kicker on its kick heading long
	// enough for the ball and the robot's kicker/dribbler to separate before a
	// normal-play role may request a completely different orientation.
	restartPostKickMinHeadingHold  = 300 * time.Millisecond
	restartPostKickMaxHeadingHold  = 600 * time.Millisecond
	restartPostKickBallClearanceMM = 300.0
)

const (
	defaultKickoffKickerID   = info.ID(1)
	defaultKickoffReceiverID = info.ID(3)
	defaultGoalieID          = info.ID(6)

	kickoffFieldMarginMM        = 300.0
	kickoffKickTargetDistanceMM = 1000.0
	kickoffPrepareDistanceMM    = 500.0
	freeKickWallDistanceMM      = 700.0
	freeKickWallSpacingMM       = 300.0
	freeKickWallRobotCount      = 3
	// freeKickSupportBallClearanceMM keeps our non-kickers clear of the ball
	// during our free kick (outside rules.RestartBallKeepout).
	freeKickSupportBallClearanceMM = 800.0
	penaltyPrepareDistanceMM       = 700.0
	penaltyBehindBallDistanceMM    = 1200.0
)

// postKickHoldComplete reports whether the kicker may stop holding its kick
// heading, measuring the ball against where the restart was taken.
func postKickHoldComplete(gi *info.GameInfo, started time.Time) bool {
	anchor, ok := gi.Status.GetGameEvent().RestartBallAnchor()
	if !ok {
		anchor = kickoffBallPosition(gi)
	}
	return restartPostKickHoldComplete(started, anchor, kickoffBallPosition(gi), gi.Now())
}

// restartPostKickHoldComplete: the hold ends after restartPostKickMaxHeadingHold,
// or after restartPostKickMinHeadingHold once the ball is
// restartPostKickBallClearanceMM from where the restart was taken.
func restartPostKickHoldComplete(
	started time.Time,
	originalBallPos info.Position,
	currentBallPos info.Position,
	now time.Time,
) bool {
	if started.IsZero() {
		return false
	}
	elapsed := now.Sub(started)
	if elapsed >= restartPostKickMaxHeadingHold {
		return true
	}
	if elapsed < restartPostKickMinHeadingHold {
		return false
	}
	return originalBallPos.Dist2d(currentBallPos) >= restartPostKickBallClearanceMM
}

func restartKickerHeading(gi *info.GameInfo, team info.Team, id info.ID) (float64, bool) {
	if gi == nil || gi.State == nil || id >= info.TEAM_SIZE {
		return 0, false
	}
	robot := gi.State.GetTeam(team)[id]
	if robot == nil {
		return 0, false
	}
	pos, err := robot.GetPosition()
	if err != nil {
		return 0, false
	}
	return pos.Angle, true
}

func holdRestartKickerHeading(
	gi *info.GameInfo,
	team info.Team,
	id info.ID,
	heading float64,
	headingOK bool,
	activityHandler *ai.ActivityHandler,
) {
	if !headingOK || gi == nil || gi.State == nil || activityHandler == nil || id >= info.TEAM_SIZE {
		return
	}
	robot := gi.State.GetTeam(team)[id]
	if robot == nil {
		return
	}
	pos, err := robot.GetPosition()
	if err != nil {
		return
	}

	// Hold translation at the measured position while retaining the heading at
	// which the restart was taken. Disabling RRT prevents a zero-distance hold
	// from acquiring a path waypoint with an unrelated orientation.
	pos.Angle = heading
	move := activity.NewMoveToPosition(team, id, pos)
	move.SetUseRRT(false)
	move.AvoidBall(false)
	activityHandler.AddActivity(move)
}

func ownGoalCenter(gi *info.GameInfo, team info.Team) info.Position {
	if gi == nil {
		return info.Position{X: ownHalfXSign(gi, team) * 4500}
	}
	return gi.OwnGoalCenter(team)
}

func robotIDInList(robotIDs []info.ID, target info.ID) bool {
	for _, robotID := range robotIDs {
		if robotID == target {
			return true
		}
	}
	return false
}

// SelectGoalieID elects the goalie: the referee-configured keeper if active,
// else defaultGoalieID, else the robot closest to the home goal. Restarts and
// the open-play planner share it so the goalie does not change hands.
func SelectGoalieID(gi *info.GameInfo, team info.Team, activeRobots []info.ID) (info.ID, bool) {
	if len(activeRobots) == 0 {
		return 0, false
	}

	if configuredID, ok := configuredGoalieID(gi, team); ok && robotIDInList(activeRobots, configuredID) {
		return configuredID, true
	}

	if robotIDInList(activeRobots, defaultGoalieID) {
		return defaultGoalieID, true
	}

	return robotClosestToHomeGoal(gi, team, activeRobots), true
}

func configuredGoalieID(gi *info.GameInfo, team info.Team) (info.ID, bool) {
	if gi == nil || gi.Status == nil {
		return 0, false
	}

	teamInfo := gi.Status.GetTeamInfo(team == info.Yellow)
	if teamInfo == nil {
		return 0, false
	}
	return info.ID(teamInfo.Goalkeeper), true
}

func robotClosestToHomeGoal(gi *info.GameInfo, team info.Team, activeRobots []info.ID) info.ID {
	if gi == nil || gi.State == nil || len(activeRobots) == 0 {
		return 0
	}

	target := kickoffGoalieHomePosition(gi, team)
	bestID := activeRobots[0]
	bestDist := math.Inf(1)
	for _, robotID := range activeRobots {
		robot := gi.State.GetTeam(team)[robotID]
		if robot == nil {
			continue
		}
		pos, err := robot.GetPosition()
		if err != nil {
			continue
		}

		dist := pos.Dist2d(target)
		if dist < bestDist {
			bestID = robotID
			bestDist = dist
		}
	}
	return bestID
}

func kickoffGoalieHomePosition(gi *info.GameInfo, team info.Team) info.Position {
	if gi != nil && gi.HasField() {
		return gi.HomeGoalDefPos(team)
	}
	return info.Position{X: ownHalfXSign(gi, team) * 4000, Y: 0, Z: 0, Angle: 0}
}

func moveGoalieToPosition(
	activeRobots []info.ID,
	team info.Team,
	gi *info.GameInfo,
	activityHandler *ai.ActivityHandler,
) (info.ID, bool) {
	goalieID, hasGoalie := SelectGoalieID(gi, team, activeRobots)
	if hasGoalie {
		activityHandler.AddActivity(activity.NewGoalie(team, goalieID))
	}
	return goalieID, hasGoalie
}

func fieldRobots(activeRobots []info.ID, goalieID info.ID, hasGoalie bool) []info.ID {
	if !hasGoalie {
		return activeRobots
	}
	return sortedRobotIDsExcluding(activeRobots, goalieID)
}

func sortedRobotIDsExcluding(robotIDs []info.ID, excludedID info.ID) []info.ID {
	filtered := make([]info.ID, 0, len(robotIDs))
	for _, robotID := range robotIDs {
		if robotID != excludedID {
			filtered = append(filtered, robotID)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i] < filtered[j]
	})
	return filtered
}

func ownHalfXSign(gi *info.GameInfo, team info.Team) float64 {
	if gi != nil {
		return gi.OwnHalfXSign(team)
	}
	// No game info (tests only): assume blue defends the negative half.
	if team == info.Blue {
		return -1
	}
	return 1
}

func kickoffBallPosition(gi *info.GameInfo) info.Position {
	if gi == nil || gi.State == nil {
		return info.Position{}
	}

	if pos, ok := gi.State.GetTrackedBall().GetFreshTrackedPosition(gi.Now(), info.TrackedStaleAfter); ok {
		return pos
	}
	if pos, err := gi.State.GetBall().GetPosition(); err == nil {
		return pos
	}
	return info.Position{}
}

func clampFloat(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}
