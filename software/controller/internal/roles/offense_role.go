package roles

import (
	"fmt"
	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"math"
	"math/rand"
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

func randVal(interval float64) float64 {
	randVal := (float64(rand.Int31())/math.Pow(2, 31)*2.0 - 1.0) * interval
	return randVal
}

func (kr *SupportAttackIntent) GetFromPosition() info.Position {

	// Some smarter way of selecting tactical positions should be done
	// here in order to help the main offensive player
	field := kr.gi.FieldSize()

	pos, _ := kr.gi.State.GetRobotPosition(kr.team, kr.id)

	if isGoalShotAvailable(kr.team, pos, kr.gi) {
		return pos
	}

	xmax := field.X
	ymax := field.Y
	for i := 0; i < 10; i++ {
		x := randVal(xmax)
		y := randVal(ymax)

		pos := info.Position{X: x, Y: y, Z: 0, Angle: 0}
		if isGoalShotAvailable(kr.team, pos, kr.gi) {
			return pos
		}
	}

	return pos
}

func (kr *SupportAttackIntent) GetTargetPosition() info.Position {
	pos, _ := kr.gi.State.GetBall().GetEstimatedPosition()

	return pos
}

func (kr *OffenseRole) Init() {
	awaitName := StateName(fmt.Sprintf("Align ID %d", kr.id))
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
