package roles

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/rules"
)

const coverYClamp = 1500.0

const (
	forwardDefenderBallOffset = 700.0
	middleDefenderBallOffset  = 1700.0
	backwardDefenderDepth     = 3200.0
	highDefenderBallOffset    = 900.0
)

var neutralHoldPos = info.Position{X: 1000, Y: 0, Z: 0, Angle: 0}

type DefenseRoleKind string

const (
	DefenseRoleCover    DefenseRoleKind = "cover"
	DefenseRoleForward  DefenseRoleKind = "forward"
	DefenseRoleMiddle   DefenseRoleKind = "middle"
	DefenseRoleBackward DefenseRoleKind = "backward"
	DefenseRoleWall     DefenseRoleKind = "wall"
	DefenseRoleHigh     DefenseRoleKind = "high"
	DefenseRoleLow      DefenseRoleKind = "low"
)

type DefenseSlot struct {
	Kind          DefenseRoleKind
	LateralOffset float64
}

func defenseXSign(gi *info.GameInfo, team info.Team) float64 {
	return gi.OwnHalfXSign(team)
}

type DefensePressState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            state_machine.StateName
	activityHandler *ai.ActivityHandler
	slot            DefenseSlot
}

func (s *DefensePressState) Initialize() {}

func (s *DefensePressState) GetName() state_machine.StateName {
	return s.name
}

func (s *DefensePressState) Update() state_machine.EventName {
	ballPos, _ := s.gi.State.GetBall().GetEstimatedPosition()
	target := DefenderTarget(s.gi, s.team, s.slot, ballPos)
	s.activityHandler.AddActivity(activity.NewMoveToPosition(s.team, s.robotId, target))
	return state_machine.NoEvent
}

// DefenderTarget is where a press-state defender in slot drives for ballPos.
// The planner uses the same function to assign robots to defender slots, so
// the robot picked for a slot is the one closest to where the slot will go.
func DefenderTarget(gi *info.GameInfo, team info.Team, slot DefenseSlot, ballPos info.Position) info.Position {
	switch slot.Kind {
	case DefenseRoleHigh:
		return ballGoalLineTarget(gi, team, ballPos, highDefenderBallOffset, slot.LateralOffset)
	case DefenseRoleLow:
		// The low defender holds the defense line, just outside our defense area.
		return depthTarget(gi, team, ballPos, rules.DefenseLineDepth(gi), slot.LateralOffset)
	case DefenseRoleForward:
		return ballGoalLineTarget(gi, team, ballPos, forwardDefenderBallOffset, slot.LateralOffset)
	case DefenseRoleMiddle:
		return ballGoalLineTarget(gi, team, ballPos, middleDefenderBallOffset, slot.LateralOffset)
	case DefenseRoleBackward:
		return depthTarget(gi, team, ballPos, backwardDefenderDepth, slot.LateralOffset)
	default:
		return depthTarget(gi, team, ballPos, backwardDefenderDepth, 0)
	}
}

// ballGoalLineTarget stands ballOffset from the ball on the ball-goal line,
// shifted lateralOffset perpendicular to it.
func ballGoalLineTarget(gi *info.GameInfo, team info.Team, ballPos info.Position, ballOffset float64, lateralOffset float64) info.Position {
	goalPos := gi.HomeGoalDefPos(team)
	toGoalX := goalPos.X - ballPos.X
	toGoalY := goalPos.Y - ballPos.Y
	dist := math.Sqrt(toGoalX*toGoalX + toGoalY*toGoalY)
	if dist < 1 {
		hold := neutralHoldPos
		hold.X *= defenseXSign(gi, team)
		hold.Angle = hold.AngleToPosition(ballPos)
		return hold
	}

	target := info.Position{
		X: ballPos.X + toGoalX/dist*ballOffset - toGoalY/dist*lateralOffset,
		Y: ballPos.Y + toGoalY/dist*ballOffset + toGoalX/dist*lateralOffset,
		Z: 0,
	}
	target = clampCoverTarget(target)
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

// depthTarget holds a fixed depth (mm from the center line toward our goal),
// tracking the ball's Y.
func depthTarget(gi *info.GameInfo, team info.Team, ballPos info.Position, depth float64, lateralOffset float64) info.Position {
	target := info.Position{
		X: defenseXSign(gi, team) * depth,
		Y: ballPos.Y + lateralOffset,
		Z: 0,
	}
	target = clampCoverTarget(target)
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func clampCoverTarget(target info.Position) info.Position {
	if target.Y > coverYClamp {
		target.Y = coverYClamp
	}
	if target.Y < -coverYClamp {
		target.Y = -coverYClamp
	}
	return target
}

type DefenseWallState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            state_machine.StateName
	activityHandler *ai.ActivityHandler
	wallPos         *info.Position
}

func (s *DefenseWallState) Initialize() {}

func (s *DefenseWallState) GetName() state_machine.StateName {
	return s.name
}

func (s *DefenseWallState) Update() state_machine.EventName {
	act := activity.NewMoveToPosition(s.team, s.robotId, *s.wallPos)
	s.activityHandler.AddActivity(act)
	return state_machine.NoEvent
}

type DefenseRole struct {
	id              info.ID
	sm              *state_machine.StateMachine
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
	pressName := state_machine.StateName(fmt.Sprintf("DefensePress ID %d", dr.id))
	wallName := state_machine.StateName(fmt.Sprintf("DefenseWall ID %d", dr.id))

	pressState := &DefensePressState{
		gi:              dr.gi,
		robotId:         dr.id,
		team:            dr.team,
		name:            pressName,
		activityHandler: dr.activityHandler,
		slot:            DefenseSlot{Kind: DefenseRoleCover},
	}

	wallState := &DefenseWallState{
		gi:              dr.gi,
		robotId:         dr.id,
		team:            dr.team,
		name:            wallName,
		activityHandler: dr.activityHandler,
		wallPos:         &dr.wallPosition,
	}

	sm := state_machine.NewStateMachine(pressState)
	sm.AddTransition(pressName, EventAttackerNear, wallState)
	sm.AddTransition(wallName, EventAttackerFar, pressState)

	dr.sm = sm
	dr.pressState = pressState
}

func (dr *DefenseRole) Run() {
	dr.sm.Update()
}

func (dr *DefenseRole) TriggerEvent(event state_machine.EventName) {
	dr.sm.TriggerEvent(event)
}

func (dr *DefenseRole) SetWallPosition(pos info.Position) {
	dr.wallPosition = pos
}

func (dr *DefenseRole) SetRoleKind(kind DefenseRoleKind) {
	if dr.pressState == nil {
		return
	}
	dr.pressState.slot = DefenseSlot{Kind: kind}
}

func (dr *DefenseRole) SetSlot(slot DefenseSlot) {
	if dr.pressState == nil {
		return
	}
	dr.pressState.slot = slot
}
