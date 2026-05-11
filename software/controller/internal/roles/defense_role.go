package roles

import (
	"fmt"
	"math"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type RoleType int

const pressMinX = 1000.0

const (
	RolePress RoleType = iota
	RoleCentral
	RoleWide
	RoleSupport
)

const (
	coverYClamp       = 2500.0
	centralCoverDepth = 2800.0
	wideCoverDepth    = 2000.0
	supportCoverDepth = 1200.0
)

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func defenseXSign(gi *info.GameInfo, team info.Team) float64 {
	isBlueTeam := team == info.Blue
	isBlueOnPositiveHalf := gi.Status.GetBlueTeamOnPositiveHalf()
	if (isBlueTeam && isBlueOnPositiveHalf) || (!isBlueTeam && !isBlueOnPositiveHalf) {
		return 1.0
	}
	return -1.0
}

type DefensePressState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            StateName
	activityHandler *ai.ActivityHandler
	role            RoleType
	yOffset         float64
	assignedTarget  info.Position
	kickActivity    *act.KickAtPosition
}

func (s *DefensePressState) Initialize() {
	clearTarget := info.Position{X: 0, Y: 0, Z: 0, Angle: 0}
	s.kickActivity = act.NewKickAtPosition(s.team, s.robotId, clearTarget)
}

func (s *DefensePressState) GetName() StateName {
	return s.name
}

func (s *DefensePressState) Update() EventName {
	enemyGoalX := s.gi.EnemyGoalCenter(s.team).X
	ownGoalX := -enemyGoalX
	sign := math.Copysign(1.0, ownGoalX)

	ballPos, _ := s.gi.State.GetBall().GetEstimatedPosition()
	robotPos, err := s.gi.State.GetRobotPosition(s.team, s.robotId)
	if err != nil {
		return "NONE"
	}

	target := s.assignedTarget
	target.Angle = target.AngleToPosition(ballPos)

	switch s.role {
	case RolePress:
		if ballPos.X*sign > 1000.0 {
			s.activityHandler.AddActivity(s.kickActivity)
		} else {
			chase := ballPos
			chase.Angle = robotPos.AngleToPosition(ballPos)
			s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, chase))
		}

	case RoleCentral:
		if target.Norm2d() < 1e-6 {
			target = info.Position{X: sign * centralCoverDepth, Y: 0, Z: 0, Angle: 0}
			target.Angle = target.AngleToPosition(ballPos)
		}
		s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, target))

	case RoleWide:
		if target.Norm2d() < 1e-6 {
			targetY := clampF(ballPos.Y+s.yOffset, -coverYClamp, coverYClamp)
			target = info.Position{X: sign * wideCoverDepth, Y: targetY, Z: 0, Angle: 0}
			target.Angle = target.AngleToPosition(ballPos)
		}
		s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, target))

	case RoleSupport:
		if target.Norm2d() < 1e-6 {
			targetY := clampF(-0.4*ballPos.Y, -coverYClamp, coverYClamp)
			target = info.Position{X: sign * supportCoverDepth, Y: targetY, Z: 0, Angle: 0}
			target.Angle = target.AngleToPosition(ballPos)
		}
		s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, target))
	}

	return "NONE"
}

type DefenseWallState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            StateName
	activityHandler *ai.ActivityHandler
	wallPos         *info.Position
}

func (s *DefenseWallState) Initialize() {}

func (s *DefenseWallState) GetName() StateName {
	return s.name
}

func (s *DefenseWallState) Update() EventName {
	s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, *s.wallPos))
	return "NONE"
}

type DefenseRole struct {
	id             info.ID
	sm             *StateMachine
	activityHandler *ai.ActivityHandler
	gi             *info.GameInfo
	team           info.Team
	wallPosition   info.Position
	pressState     *DefensePressState
}

func NewDefenseRole(robotID info.ID, activityHandler ai.ActivityHandler, gi *info.GameInfo, team info.Team) *DefenseRole {
	return &DefenseRole{
		id:              robotID,
		sm:              nil,
		activityHandler: &activityHandler,
		gi:              gi,
		team:            team,
	}
}

func (dr *DefenseRole) Init() {
	pressName := StateName(fmt.Sprintf("DefensePress ID %d", dr.id))
	wallName := StateName(fmt.Sprintf("DefenseWall ID %d", dr.id))

	pressState := &DefensePressState{
		gi:              dr.gi,
		robotId:         dr.id,
		team:            dr.team,
		name:            pressName,
		activityHandler: dr.activityHandler,
		role:            RoleSupport,
		yOffset:         0,
		assignedTarget:  info.Position{},
	}
	dr.pressState = pressState

	wallState := &DefenseWallState{
		gi:              dr.gi,
		robotId:         dr.id,
		team:            dr.team,
		name:            wallName,
		activityHandler: dr.activityHandler,
		wallPos:         &dr.wallPosition,
	}

	sm := NewStateMachine(pressState)
	sm.AddTransition(pressName, "ATTACKER_NEAR", wallState)
	sm.AddTransition(wallName, "ATTACKER_FAR", pressState)

	dr.sm = sm
}

func (dr *DefenseRole) SetRole(r RoleType, yOffset float64) {
	dr.pressState.role = r
	dr.pressState.yOffset = yOffset
}

func (dr *DefenseRole) GetRole() (RoleType, float64) {
	return dr.pressState.role, dr.pressState.yOffset
}

func (dr *DefenseRole) SetAssignedTarget(pos info.Position) {
	dr.pressState.assignedTarget = pos
}

func (dr *DefenseRole) Run() {
	dr.sm.Update()
}

func (dr *DefenseRole) TriggerEvent(event EventName) {
	dr.sm.TriggerEvent(event)
}

func (dr *DefenseRole) SetWallPosition(pos info.Position) {
	dr.wallPosition = pos
}