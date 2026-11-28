package roles

import (
	"fmt"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const GoalieBallControlRadius = 300.0

type GoalieRole struct {
	id              info.ID
	sm              *state_machine.StateMachine
	activityHandler *ai.ActivityHandler
	gi              *info.GameInfo
	team            info.Team
	clearFallback   info.Position
	deadBall        deadBallTracker
	safeClear       *GoalieSafeClearIntent
	collectName     state_machine.StateName
	alignName       state_machine.StateName
	kickName        state_machine.StateName
}

func NewGoalieRole(robotID info.ID, activityHandler ai.ActivityHandler, team info.Team, clearTarget info.Position) *GoalieRole {
	return &GoalieRole{
		id:              robotID,
		sm:              nil,
		activityHandler: &activityHandler,
		gi:              &info.GameInfo{},
		team:            team,
		clearFallback:   clearTarget,
	}
}

// SetClearFallback updates where the goalie clears to when no safe pass
// target exists (it flips with the field side at half time).
func (gr *GoalieRole) SetClearFallback(pos info.Position) {
	gr.clearFallback = pos
	if gr.safeClear != nil {
		gr.safeClear.fallback = pos
	}
}

func (gr *GoalieRole) SetGameInfo(gi info.GameInfo) {
	*gr.gi = gi
	gr.deadBall.Observe(gr.gi, gr.team)
}

func (gr *GoalieRole) HasBallControl(radius float64) bool {
	robotPos, err := gr.gi.State.GetRobotPosition(gr.team, gr.id)
	if err != nil {
		return false
	}

	ballPos, err := gr.gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return false
	}

	return robotPos.Dist2d(ballPos) <= radius
}

func (gr *GoalieRole) ShouldClearBall(radius float64, minDefendedX float64) bool {
	if !gr.HasBallControl(radius) {
		return false
	}

	ballPos, err := gr.gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return false
	}

	return defenseXSign(gr.gi, gr.team)*ballPos.X > minDefendedX
}

func (gr *GoalieRole) Init() {
	defendName := state_machine.StateName(fmt.Sprintf("GoalieDefend ID %d", gr.id))
	collectName := state_machine.StateName(fmt.Sprintf("GoalieCollectDeadBall ID %d", gr.id))
	alignName := state_machine.StateName(fmt.Sprintf("GoalieAlignClear ID %d", gr.id))
	kickName := state_machine.StateName(fmt.Sprintf("GoalieKickClear ID %d", gr.id))
	gr.collectName = collectName
	gr.alignName = alignName
	gr.kickName = kickName

	clearContext := &GoalieSafeClearIntent{
		gi:       gr.gi,
		team:     gr.team,
		selfID:   gr.id,
		fallback: gr.clearFallback,
	}
	gr.safeClear = clearContext

	defend := &GoalieDefendState{
		gi:              gr.gi,
		team:            gr.team,
		robotId:         gr.id,
		name:            defendName,
		activityHandler: gr.activityHandler,
	}
	collect := &GoalieCollectDeadBallState{
		gi:              gr.gi,
		team:            gr.team,
		robotId:         gr.id,
		name:            collectName,
		activityHandler: gr.activityHandler,
	}
	align := &GoalieSafeAlignState{
		Ctx:             clearContext,
		Gi:              gr.gi,
		Team:            gr.team,
		RobotId:         gr.id,
		Name:            alignName,
		ActivityHandler: gr.activityHandler,
	}
	kick := &GoalieSafeKickState{
		Ctx:             clearContext,
		Name:            kickName,
		Gi:              gr.gi,
		Team:            gr.team,
		RobotId:         gr.id,
		ActivityHandler: gr.activityHandler,
	}

	sm := state_machine.NewStateMachine(defend)
	sm.AddTransition(defendName, EventDeadBallTrapped, collect)
	sm.AddTransition(defendName, EventBallOwner, align)
	sm.AddTransition(collectName, EventBallOwner, align)
	sm.AddTransition(collectName, EventBallLost, defend)
	sm.AddTransition(alignName, EventAligned, kick)
	sm.AddTransition(alignName, EventBallLost, defend)
	sm.AddTransition(kickName, EventKicked, defend)
	sm.AddTransition(kickName, EventMisaligned, align)
	sm.AddTransition(kickName, EventBallLost, defend)

	gr.sm = sm
}

func (gr *GoalieRole) Run() {
	gr.sm.Update()
}

func (gr *GoalieRole) IsDeadBallRescueActive() bool {
	if gr == nil || gr.sm == nil {
		return false
	}
	stateName := gr.sm.CurrentStateName()
	return stateName == gr.collectName || stateName == gr.alignName || stateName == gr.kickName
}

func (gr *GoalieRole) TriggerEvent(event state_machine.EventName) {
	if event == EventBallLost && gr.safeClear != nil {
		gr.safeClear.ResetTarget()
	}
	gr.sm.TriggerEvent(event)
}
