package roles

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const GoalieBallControlRadius = 300.0

type GoalieRole struct {
	id              info.ID
	sm              *StateMachine
	activityHandler *ai.ActivityHandler
	gi              *info.GameInfo
	team            info.Team
	clearFallback   info.Position
}

type GoalieClearIntent struct {
	gi       *info.GameInfo
	team     info.Team
	selfID   info.ID
	fallback info.Position
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

func (gr *GoalieRole) SetGameInfo(gi info.GameInfo) {
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

func (gc *GoalieClearIntent) GetTargetPosition() info.Position {
	ballPos, err := gc.gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return gc.fallback
	}

	teamRobots := gc.gi.State.GetTeam(gc.team)
	bestDist := 0.0
	var bestPos info.Position
	found := false

	for id, robot := range teamRobots {
		if info.ID(id) == gc.selfID || robot == nil {
			continue
		}

		pos, err := robot.GetPosition()
		if err != nil {
			continue
		}

		dist := pos.Dist2d(ballPos)
		if !found || dist < bestDist {
			bestDist = dist
			bestPos = pos
			found = true
		}
	}

	if found {
		return bestPos
	}

	return gc.fallback
}

func (gc *GoalieClearIntent) GetFromPosition() info.Position {
	pos, _ := gc.gi.State.GetBall().GetEstimatedPosition()
	return pos
}

func (gr *GoalieRole) Init() {
	defendName := StateName(fmt.Sprintf("GoalieDefend ID %d", gr.id))
	alignName := StateName(fmt.Sprintf("GoalieAlignClear ID %d", gr.id))
	kickName := StateName(fmt.Sprintf("GoalieKickClear ID %d", gr.id))

	clearContext := GoalieClearIntent{
		gi:       gr.gi,
		team:     gr.team,
		selfID:   gr.id,
		fallback: gr.clearFallback,
	}

	defend := &GoalieDefendState{
		gi:              gr.gi,
		team:            gr.team,
		robotId:         gr.id,
		name:            defendName,
		activityHandler: gr.activityHandler,
	}
	align := &AlignState{
		Ctx:             &clearContext,
		Gi:              gr.gi,
		Team:            gr.team,
		RobotId:         gr.id,
		Name:            alignName,
		ActivityHandler: gr.activityHandler,
	}
	kick := &KickState{
		Ctx:             &clearContext,
		Name:            kickName,
		Gi:              gr.gi,
		Team:            gr.team,
		RobotId:         gr.id,
		ActivityHandler: gr.activityHandler,
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
