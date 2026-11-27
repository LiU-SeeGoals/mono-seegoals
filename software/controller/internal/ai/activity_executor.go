package ai

import (
	"fmt"
	"sync"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/logger"
)

type activityExecutor struct {
	activities   *[info.TEAM_SIZE]activity.Activity
	activityLock *sync.Mutex
	constraints  motionConstraints
	frameMonitor *helper.FrameSkipMonitor
}

func newActivityExecutor(team info.Team, activities *[info.TEAM_SIZE]activity.Activity, lock *sync.Mutex) *activityExecutor {
	fb := &activityExecutor{
		activities:   activities,
		activityLock: lock,
		constraints: motionConstraints{
			team:                   team,
			restrictToOpponentHalf: restrictsTeamToOpponentHalf(config.RestrictToOpponentHalfTeam(), team),
		},
		frameMonitor: helper.NewFrameSkipMonitor(fmt.Sprintf("%s activity_executor", team)),
	}
	if fb.constraints.restrictToOpponentHalf {
		logger.Logger.Warnf("%v robots are restricted to the opponent half", team)
	}
	return fb
}

// execute turns the current activities into rule-checked actions, in robot
// ID order.
func (fb *activityExecutor) execute(gi *info.GameInfo) []action.Action {
	fb.frameMonitor.Observe(gi.VisionFrame())

	// HALT is a hard safety boundary: an old kick activity must never produce
	// a command from a halted frame.
	if actions, halted := fb.constraints.beginFrame(gi); halted {
		return actions
	}

	fb.activityLock.Lock()
	current := *fb.activities
	fb.activityLock.Unlock()

	actions := make([]action.Action, 0, info.TEAM_SIZE)
	for _, act := range current {
		if act == nil {
			continue
		}
		actions = append(actions, fb.constraints.apply(act.GetAction(gi), gi))
	}
	return actions
}
