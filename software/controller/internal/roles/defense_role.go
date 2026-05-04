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
	RolePress   RoleType = iota // closest to ball — actively presses
	RoleCentral                  // second — covers ball-to-goal line
	RoleWide                     // third/fourth — wide flank coverage
	RoleSupport                  // remaining — midfield cover
)

const (
	coverYClamp       = 2500.0 // mm — ~0.8× half field width
	centralCoverDepth = 2800.0 // mm from own goal along ball-goal line
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
	// yOffset is fixed per rank so bots never stack — +600 left wing, -600 right wing, 0 center
	yOffset      float64
	kickActivity *act.KickAtPosition
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
	ownGoalCenter := info.Position{X: ownGoalX, Y: 0, Z: 0, Angle: 0}

	robotPos, err := s.gi.State.GetRobotPosition(s.team, s.robotId)
	if err != nil {
		return "NONE"
	}

	switch s.role {
	case RolePress:
		// Kick clear when ball enters our half, otherwise chase ball
		if ballPos.X*sign > 1000.0 {
			s.activityHandler.AddActivity(s.kickActivity)
		} else {
			target := ballPos
			target.Angle = robotPos.AngleToPosition(ballPos)
			s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, target))
		}

	case RoleCentral:
		// Position on line from own goal toward ball at centralCoverDepth from goal
		goalToBall := ballPos.Sub(&ownGoalCenter)
		if goalToBall.Norm2d() < 1.0 {
			target := info.Position{X: ownGoalX - sign*centralCoverDepth, Y: 0, Z: 0, Angle: 0}
			target.Angle = target.AngleToPosition(ballPos)
			s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, target))
		} else {
			norm := goalToBall.Normalize2d()
			scaled := norm.Scale(centralCoverDepth)
			target := ownGoalCenter.Add(&scaled)
			target.Angle = target.AngleToPosition(ballPos)
			s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, target))
		}

	case RoleWide:
		// Fixed Y offset from ball — rank determines which wing this bot covers
		targetY := clampF(ballPos.Y+s.yOffset, -coverYClamp, coverYClamp)
		target := info.Position{X: sign * wideCoverDepth, Y: targetY, Z: 0, Angle: 0}
		target.Angle = target.AngleToPosition(ballPos)
		s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, target))

	case RoleSupport:
		// Fixed Y offset from ball at midfield depth
		targetY := clampF(ballPos.Y+s.yOffset, -coverYClamp, coverYClamp)
		target := info.Position{X: sign * supportCoverDepth, Y: targetY, Z: 0, Angle: 0}
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
	s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, *s.wallPos))
	return "NONE"
}

type DefenseRole struct {
	id              info.ID
	sm              *StateMachine
	activityHandler *ai.ActivityHandler
	gi              *info.GameInfo
	team            info.Team
	wallPosition    info.Position
	pressState      *DefensePressState
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

// SetRole assigns the bot's role and Y offset for this frame.
// yOffset is fixed per rank so bots on the same line don't stack.
func (dr *DefenseRole) SetRole(r RoleType, yOffset float64) {
	dr.pressState.role = r
	dr.pressState.yOffset = yOffset
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
