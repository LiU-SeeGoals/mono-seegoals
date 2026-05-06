package roles

import (
	"fmt"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const coverYClamp = 1500.0

const pressMinX = 1000.0

var neutralHoldPos = info.Position{X: 1000, Y: 0, Z: 0, Angle: 0}

var clearTarget = info.Position{X: 0, Y: 0, Z: 0, Angle: 0}

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
	isPrimary       bool
	coverDepth      float64
	kickActivity    *act.KickAtPosition
}

func (s *DefensePressState) Initialize() {
	if s.isPrimary {
		s.kickActivity = act.NewKickAtPosition(s.team, s.robotId, clearTarget)
	}
}

func (s *DefensePressState) GetName() StateName {
	return s.name
}

func (s *DefensePressState) Update() EventName {
	xSign := defenseXSign(s.gi, s.team)

	if s.isPrimary {
		ballPos, _ := s.gi.State.GetBall().GetEstimatedPosition()

		if xSign*ballPos.X < pressMinX {
			hold := neutralHoldPos
			hold.X *= xSign
			hold.Angle = hold.AngleToPosition(ballPos) // face the ball
			s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, hold))
		} else {
			s.activityHandler.AddActivity(s.kickActivity)
		}
	} else {
		ballPos, _ := s.gi.State.GetBall().GetEstimatedPosition()
		target := info.Position{X: xSign * s.coverDepth, Y: ballPos.Y, Z: 0, Angle: 0}
		if target.Y > coverYClamp {
			target.Y = coverYClamp
		}
		if target.Y < -coverYClamp {
			target.Y = -coverYClamp
		}
		target.Angle = target.AngleToPosition(ballPos)
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
	activity := act.NewMoveToPosition(s.team, s.robotId, *s.wallPos)
	s.activityHandler.AddActivity(activity)
	return "NONE"
}

type DefenseRole struct {
	id              info.ID
	sm              *StateMachine
	activityHandler *ai.ActivityHandler
	gi              *info.GameInfo
	team            info.Team
	wallPosition    info.Position
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

	isPrimary := dr.id == 5
	coverDepth := 3200.0
	if dr.id == 7 {
		coverDepth = 3700.0
	}

	pressState := &DefensePressState{
		gi:              dr.gi,
		robotId:         dr.id,
		team:            dr.team,
		name:            pressName,
		activityHandler: dr.activityHandler,
		isPrimary:       isPrimary,
		coverDepth:      coverDepth,
	}

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

func (dr *DefenseRole) Run() {
	dr.sm.Update()
}

func (dr *DefenseRole) TriggerEvent(event EventName) {
	dr.sm.TriggerEvent(event)
}

func (dr *DefenseRole) SetWallPosition(pos info.Position) {
	dr.wallPosition = pos
}
