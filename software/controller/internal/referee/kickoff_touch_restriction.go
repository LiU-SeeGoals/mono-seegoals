package referee

import (
	"time"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	sslRobotRadiusMM            = 90.0
	robotBallContactToleranceMM = 10.0
	maxRobotHeightMM            = 150.0
	ballTouchRobotTimeout       = time.Second
)

type kickoffTouchRestriction struct {
	team   info.Team
	id     info.ID
	active bool
}

func (r *kickoffTouchRestriction) arm(team info.Team, id info.ID) {
	r.team = team
	r.id = id
	r.active = true
}

func (r *kickoffTouchRestriction) clear() {
	r.active = false
}

func (r *kickoffTouchRestriction) robot() (info.ID, bool) {
	return r.id, r.active
}

func (r *kickoffTouchRestriction) update(gi *info.GameInfo) {
	if !r.active || gi == nil || gi.State == nil {
		return
	}

	if anotherRobotTouchesBall(gi, r.team, r.id) {
		r.clear()
	}
}

func anotherRobotTouchesBall(gi *info.GameInfo, restrictedTeam info.Team, restrictedID info.ID) bool {
	ballPos, ok := contactBallPosition(gi)
	if !ok || ballPos.Z > maxRobotHeightMM {
		return false
	}

	contactDistance := sslRobotRadiusMM + info.BallRadius + robotBallContactToleranceMM
	for _, team := range []info.Team{info.Blue, info.Yellow} {
		robots := gi.State.GetTeam(team)
		for id, robot := range robots {
			robotID := info.ID(id)
			if robot == nil || !robot.IsActive() || (team == restrictedTeam && robotID == restrictedID) {
				continue
			}

			robotPos, timestamp, err := robot.GetPositionTime()
			if err != nil || robotObservationExpired(gi, timestamp) {
				continue
			}
			if robotPos.Dist2d(ballPos) <= contactDistance {
				return true
			}
		}
	}

	return false
}

func contactBallPosition(gi *info.GameInfo) (info.Position, bool) {
	if gi == nil || gi.State == nil {
		return info.Position{}, false
	}
	if pos, ok := gi.State.GetTrackedBall().GetTrackedPosition(); ok {
		return pos, true
	}
	pos, err := gi.State.GetBall().GetEstimatedPosition()
	return pos, err == nil
}

func robotObservationExpired(gi *info.GameInfo, robotTimestamp int64) bool {
	frameTimestamp := gi.State.GetMessageReceivedTime()
	if frameTimestamp <= 0 || robotTimestamp <= 0 || frameTimestamp < robotTimestamp {
		return false
	}
	return time.Duration(frameTimestamp-robotTimestamp)*time.Millisecond > ballTouchRobotTimeout
}
