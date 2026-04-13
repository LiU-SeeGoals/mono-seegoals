package roles

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

const GoalieBallControlRadius = 300.0

type GoalieRole struct {
	id              info.ID
	sm              *StateMachine
	activityHandler *ai.ActivityHandler
	gi              *GameInfo
	team            Team
	clearTarget     info.Position
}

type GoalieClearIntent struct {
	gi     *GameInfo
	target info.Position
}

func NewGoalieRole(robotID ID, activityHandler ai.ActivityHandler, team Team, clearTarget info.Position) *GoalieRole {
	return &GoalieRole{
		id:              robotID,
		sm:              nil,
		activityHandler: &activityHandler,
		gi:              &GameInfo{},
		team:            team,
		clearTarget:     clearTarget,
	}
}

func (gr *GoalieRole) SetGameInfo(gi GameInfo) {
	*gr.gi = gi
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

func (gc *GoalieClearIntent) getTargetPosition() info.Position {
	return gc.target
}

func (gc *GoalieClearIntent) getFromPosition() info.Position {
	pos, _ := gc.gi.State.GetBall().GetEstimatedPosition()
	return pos
}

func (gr *GoalieRole) Init() {
	defendName := StateName(fmt.Sprintf("GoalieDefend ID %d", gr.id))
	alignName := StateName(fmt.Sprintf("GoalieAlignClear ID %d", gr.id))
	kickName := StateName(fmt.Sprintf("GoalieKickClear ID %d", gr.id))

	clearContext := GoalieClearIntent{
		gi:     gr.gi,
		target: gr.clearTarget,
	}

	defend := &GoalieDefendState{
		gi:              gr.gi,
		team:            gr.team,
		robotId:         gr.id,
		name:            defendName,
		activityHandler: gr.activityHandler,
	}
	align := &AlignState{
		ctx:             &clearContext,
		gi:              gr.gi,
		team:            gr.team,
		robotId:         gr.id,
		name:            alignName,
		activityHandler: gr.activityHandler,
	}
	kick := &KickState{
		ctx:     &clearContext,
		name:    kickName,
		gi:      gr.gi,
		team:    gr.team,
		robotId: gr.id,
		handle:  gr.activityHandler,
	}

	sm := NewStateMachine(defend)
	sm.AddTransition(defendName, "BALL_OWNER", align)
	sm.AddTransition(alignName, "ALIGNED", kick)
	sm.AddTransition(alignName, "BALL_LOST", defend)
	sm.AddTransition(kickName, "KICKED", defend)
	sm.AddTransition(kickName, "BALL_LOST", defend)

	gr.sm = sm
}

func (gr *GoalieRole) Run() {
	gr.sm.Update()
}

func (gr *GoalieRole) TriggerEvent(event EventName) {
	gr.sm.TriggerEvent(event)
}
