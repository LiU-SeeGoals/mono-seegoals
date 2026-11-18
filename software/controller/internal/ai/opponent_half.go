package ai

import (
	"strings"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const opponentHalfRobotRadiusMM = 90.0

func restrictsTeamToOpponentHalf(setting string, team info.Team) bool {
	return strings.EqualFold(strings.TrimSpace(setting), team.String())
}

func opponentHalfRestrictionActive(gi *info.GameInfo) bool {
	if gi == nil || gi.Status == nil {
		return true
	}
	event := gi.Status.GetGameEvent()
	if event == nil {
		return true
	}
	switch event.CurrentState {
	case info.STATE_KICKOFF_PREPARATION, info.STATE_PENALTY_PREPARATION, info.STATE_BALL_PLACEMENT:
		return false
	}
	return true
}

func clampActionToOpponentHalf(act action.Action, gi *info.GameInfo, team info.Team) action.Action {
	move, ok := act.(*action.MoveTo)
	if !ok || move == nil || !opponentHalfRestrictionActive(gi) {
		return act
	}

	sign := -gi.OwnHalfXSign(team)
	allowed := func(pos info.Position) bool {
		return sign*pos.X >= opponentHalfRobotRadiusMM
	}

	if !allowed(move.Dest) {
		move.Dest.X = sign * opponentHalfRobotRadiusMM
		move.KickSpeed = 0
		move.SimKickSpeed = 0
	}

	if len(move.Path) > 0 {
		leaving := !allowed(move.Pos)
		path := make([]info.Position, 0, len(move.Path)+1)
		truncated := false
		for _, waypoint := range move.Path {
			if allowed(waypoint) {
				leaving = false
			} else if !leaving {
				truncated = true
				break
			}
			path = append(path, waypoint)
		}
		if truncated {
			path = append(path, move.Dest)
		}
		move.Path = path
	}

	return move
}
