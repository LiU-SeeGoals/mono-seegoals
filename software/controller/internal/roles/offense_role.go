package roles

import (
	"fmt"
	"math"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

type OffenseRole struct {
	id              info.ID
	sm              *StateMachine
	activityHandler *ai.ActivityHandler
	gi              *GameInfo
	team            Team
}

func NewOffenseRole(robotID ID, activityHandler ai.ActivityHandler, gi *GameInfo, team Team) *OffenseRole {
	return &OffenseRole{
		id:              robotID,
		sm:              nil,
		activityHandler: &activityHandler,
		gi:              gi,
		team:            team,
	}
}

func (kr *AttemptGoalIntent) GetBestHomiePos() info.Position {

	// TODO implement some smarter way of kicking to best homie
	// For example one that has free sight of the goal
	homieId := 1
	if kr.id == 1 {
		homieId = 3
	}

	pos, err := kr.gi.State.GetRobotPosition(kr.team, ID(homieId))
	if err != nil {
		fmt.Println("Could not get best homies pos")
	}

	return pos
}

type AttemptGoalIntent struct {
	gi   *GameInfo
	team Team
	id   ID
}

func isGoalShotAvailable(team info.Team, from info.Position, gi *GameInfo) bool {

	// Try to shoot goal if the sight is clear
	// Otherwise pass to a homie

	goalPosition := gi.EnemyGoalCenter(team)

	enemies := gi.State.GetOtherTeam(team)
	for i := 0; i < int(TEAM_SIZE); i++ {
		enemyPos, err := enemies[i].GetPosition()
		if err != nil {
			continue
		}
		dist := info.DistToLineSegment(goalPosition.ToV2(), from.ToV2(), enemyPos.ToV2())
		if dist < 200 {
			return false
		}
	}

	return true
}

func (kr *AttemptGoalIntent) GetTargetPosition() info.Position {

	// Try to shoot goal if the sight is clear
	// Otherwise pass to a homie
	goalPosition := kr.gi.EnemyGoalCenter(kr.team)
	robotPos, err := kr.gi.State.GetRobotPosition(kr.team, kr.id)

	if err != nil {
		fmt.Println("failed robot pos")
	}

	if isGoalShotAvailable(kr.team, robotPos, kr.gi) {
		return goalPosition
	} else {
		return kr.GetBestHomiePos()
	}
}

func (kr *AttemptGoalIntent) GetFromPosition() info.Position {
	pos, _ := kr.gi.State.GetBall().GetEstimatedPosition()

	return pos
}

type SupportAttackIntent struct {
	gi   *GameInfo
	team Team
	id   ID
}

type supportOffset struct {
	forward float64
	lateral float64
}

var supportOffsets = []supportOffset{
	{forward: 900, lateral: -900},
	{forward: 900, lateral: 900},
	{forward: 1500, lateral: -1400},
	{forward: 1500, lateral: 1400},
	{forward: 2200, lateral: -500},
	{forward: 2200, lateral: 500},
}

func clamp(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

func (kr *SupportAttackIntent) supportCandidate(ballPos, goalPos info.Position, offset supportOffset) info.Position {
	field := kr.gi.FieldSize()
	margin := 400.0
	halfX := field.X/2 - margin
	halfY := field.Y/2 - margin

	toGoalX := goalPos.X - ballPos.X
	toGoalY := goalPos.Y - ballPos.Y
	dist := math.Sqrt(toGoalX*toGoalX + toGoalY*toGoalY)
	if dist < 1 {
		pos, _ := kr.gi.State.GetRobotPosition(kr.team, kr.id)
		return pos
	}

	forwardX := toGoalX / dist
	forwardY := toGoalY / dist
	lateralX := -forwardY
	lateralY := forwardX

	x := ballPos.X + forwardX*offset.forward + lateralX*offset.lateral
	y := ballPos.Y + forwardY*offset.forward + lateralY*offset.lateral
	target := info.Position{
		X: x,
		Y: y,
		Z: 0,
	}
	target.X = clamp(target.X, -halfX, halfX)
	target.Y = clamp(target.Y, -halfY, halfY)
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (kr *SupportAttackIntent) GetFromPosition() info.Position {

	// Some smarter way of selecting tactical positions should be done
	// here in order to help the main offensive player
	currentPos, err := kr.gi.State.GetRobotPosition(kr.team, kr.id)
	if err != nil {
		return info.Position{}
	}

	ballPos, err := kr.gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return currentPos
	}

	goalPos := kr.gi.EnemyGoalCenter(kr.team)
	slot := int(kr.id) % len(supportOffsets)
	fallback := kr.supportCandidate(ballPos, goalPos, supportOffsets[slot])

	for i := 0; i < len(supportOffsets); i++ {
		offset := supportOffsets[(slot+i)%len(supportOffsets)]
		candidate := kr.supportCandidate(ballPos, goalPos, offset)
		if isGoalShotAvailable(kr.team, candidate, kr.gi) {
			return candidate
		}
	}

	return fallback
}

func (kr *SupportAttackIntent) GetTargetPosition() info.Position {
	pos, _ := kr.gi.State.GetBall().GetEstimatedPosition()

	return pos
}

func (kr *OffenseRole) Init() {
	awaitName := StateName(fmt.Sprintf("Support ID %d", kr.id))
	kickPrepareName := StateName(fmt.Sprintf("KickPrepare ID %d", kr.id))
	kickName := StateName(fmt.Sprintf("Kick ID %d", kr.id))

	offenseContext := AttemptGoalIntent{gi: kr.gi, team: kr.team, id: kr.id}
	supportContext := SupportAttackIntent{gi: kr.gi, team: kr.team, id: kr.id}

	awaitBall := &SupportState{ctx: &supportContext, gi: kr.gi, team: kr.team, robotId: kr.id, name: awaitName, activityHandler: kr.activityHandler}
	prepareKick := &AlignState{Ctx: &offenseContext, Gi: kr.gi, Team: kr.team, RobotId: kr.id, Name: kickPrepareName, ActivityHandler: kr.activityHandler}
	kick := &KickState{Ctx: &offenseContext, Name: kickName, Gi: kr.gi, Team: kr.team, RobotId: kr.id, ActivityHandler: kr.activityHandler}

	sm := NewStateMachine(awaitBall)

	sm.AddTransition(awaitName, "BALL_OWNER", prepareKick)

	sm.AddTransition(kickPrepareName, "ALIGNED", kick)
	sm.AddTransition(kickPrepareName, "BALL_LOST", awaitBall)
	sm.AddTransition(kickName, "KICKED", prepareKick)
	sm.AddTransition(kickName, "BALL_LOST", awaitBall)

	kr.sm = sm
}

func (kr *OffenseRole) Run() {
	kr.sm.Update()
}

func (kr *OffenseRole) TriggerEvent(event EventName) {
	kr.sm.TriggerEvent(event)
}
