package roles

import (
	"fmt"
	"math"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const coverYClamp = 1500.0

const (
	forwardDefenderBallOffset = 700.0
	middleDefenderBallOffset  = 1700.0
	backwardDefenderDepth     = 3200.0
	highDefenderBallOffset    = 900.0
	lowDefenderDepth          = 3600.0
	defensePossessionDistance = 200.0
	dangerousOpponentGoalDist = 2500
	coverReceiverWeight       = 2.0 / 3.0
)

var neutralHoldPos = info.Position{X: 1000, Y: 0, Z: 0, Angle: 0}
var coverParkingPos = info.Position{X: -4000, Y: -4000, Z: 0, Angle: 0}

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
	slot            DefenseSlot
}

func (s *DefensePressState) Initialize() {}

func (s *DefensePressState) GetName() StateName {
	return s.name
}

func (s *DefensePressState) Update() EventName {
	xSign := defenseXSign(s.gi, s.team)

	ballPos, _ := s.gi.State.GetBall().GetEstimatedPosition()
	var target info.Position
	switch s.slot.Kind {
	case DefenseRoleHigh:
		target = s.ballGoalLineTarget(ballPos, highDefenderBallOffset, s.slot.LateralOffset)
	case DefenseRoleLow:
		target = s.lowDefenderTarget(ballPos, s.slot.LateralOffset)
	case DefenseRoleForward:
		target = s.ballGoalLineTarget(ballPos, forwardDefenderBallOffset, s.slot.LateralOffset)
	case DefenseRoleMiddle:
		target = s.ballGoalLineTarget(ballPos, middleDefenderBallOffset, s.slot.LateralOffset)
	case DefenseRoleBackward:
		target = s.backwardDefenderTarget(ballPos, s.slot.LateralOffset)
	case DefenseRoleCover:
		target = s.coverTarget(ballPos)
	default:
		target = info.Position{X: xSign * backwardDefenderDepth, Y: ballPos.Y, Z: 0, Angle: 0}
		target = clampCoverTarget(target)
		target.Angle = target.AngleToPosition(ballPos)
	}

	s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, target))
	return "NONE"
}

func (s *DefensePressState) ballGoalLineTarget(ballPos info.Position, ballOffset float64, lateralOffset float64) info.Position {
	goalPos := s.gi.HomeGoalDefPos(s.team)
	toGoalX := goalPos.X - ballPos.X
	toGoalY := goalPos.Y - ballPos.Y
	dist := math.Sqrt(toGoalX*toGoalX + toGoalY*toGoalY)
	if dist < 1 {
		hold := neutralHoldPos
		hold.X *= defenseXSign(s.gi, s.team)
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

func (s *DefensePressState) backwardDefenderTarget(ballPos info.Position, lateralOffset float64) info.Position {
	target := info.Position{
		X: defenseXSign(s.gi, s.team) * backwardDefenderDepth,
		Y: ballPos.Y + lateralOffset,
		Z: 0,
	}
	target = clampCoverTarget(target)
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (s *DefensePressState) lowDefenderTarget(ballPos info.Position, lateralOffset float64) info.Position {
	target := info.Position{
		X: defenseXSign(s.gi, s.team) * lowDefenderDepth,
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

func (s *DefensePressState) coverTarget(ballPos info.Position) info.Position {
	ballPossessor, possessorPos, found := s.opponentWithBall(ballPos)
	var excludedOpponent *info.Robot
	if found {
		excludedOpponent = ballPossessor
		fmt.Printf("Opponent with ball: team=%s id=%d position=(%.0f, %.0f)\n",
			ballPossessor.GetTeam(), ballPossessor.GetID(), possessorPos.X, possessorPos.Y)
	}

	dangerousOpponent, dangerousPos, goalDistance, found := s.closestOpponentToOwnGoalExcept(excludedOpponent)
	if !found || goalDistance > dangerousOpponentGoalDist {
		return coverParkingPos
	}
	fmt.Printf("Dangerous opponent closest to own goal: team=%s id=%d position=(%.0f, %.0f)\n",
		dangerousOpponent.GetTeam(), dangerousOpponent.GetID(), dangerousPos.X, dangerousPos.Y)

	target := info.Position{
		X: ballPos.X + (dangerousPos.X-ballPos.X)*coverReceiverWeight,
		Y: ballPos.Y + (dangerousPos.Y-ballPos.Y)*coverReceiverWeight,
		Z: 0,
	}
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (s *DefensePressState) closestOpponentToOwnGoal() (*info.Robot, info.Position, bool) {
	opponent, position, _, found := s.closestOpponentToOwnGoalExcept(nil)
	return opponent, position, found
}

func (s *DefensePressState) closestOpponentToOwnGoalExcept(excluded *info.Robot) (*info.Robot, info.Position, float64, bool) {
	if s.gi == nil || s.gi.State == nil {
		return nil, info.Position{}, 0, false
	}

	halfFieldLength := 4500.0
	if s.gi.HasField() {
		fieldSize := s.gi.FieldSize()
		if fieldSize.X > 0 {
			halfFieldLength = fieldSize.X / 2
		}
	}
	goalPos := info.Position{
		X: s.gi.OwnHalfXSign(s.team) * halfFieldLength,
		Y: 0,
		Z: 0,
	}

	opponents := s.gi.State.GetOtherTeam(s.team)
	var closestOpponent *info.Robot
	var closestOpponentPos info.Position
	closestDistance := math.Inf(1)

	for _, opponent := range opponents {
		if opponent == nil || !opponent.IsActive() {
			continue
		}
		if excluded != nil && opponent.GetTeam() == excluded.GetTeam() && opponent.GetID() == excluded.GetID() {
			continue
		}

		opponentPos, err := opponent.GetPosition()
		if err != nil {
			continue
		}

		distance := opponentPos.Dist2d(goalPos)
		if distance < closestDistance {
			closestOpponent = opponent
			closestOpponentPos = opponentPos
			closestDistance = distance
		}
	}

	return closestOpponent, closestOpponentPos, closestDistance, closestOpponent != nil
}

func (s *DefensePressState) opponentWithBall(ballPos info.Position) (*info.Robot, info.Position, bool) {
	if s.gi == nil || s.gi.State == nil {
		return nil, info.Position{}, false
	}

	opponents := s.gi.State.GetOtherTeam(s.team)
	var closestOpponent *info.Robot
	var closestOpponentPos info.Position
	closestDistance := math.Inf(1)

	for _, opponent := range opponents {
		if opponent == nil || !opponent.IsActive() {
			continue
		}

		opponentPos, err := opponent.GetPosition()
		if err != nil {
			continue
		}

		distance := opponentPos.Dist2d(ballPos)
		if distance < closestDistance {
			closestOpponent = opponent
			closestOpponentPos = opponentPos
			closestDistance = distance
		}
	}

	if closestOpponent == nil || closestDistance > defensePossessionDistance {
		return nil, info.Position{}, false
	}

	return closestOpponent, closestOpponentPos, true
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
	coverName := StateName(fmt.Sprintf("DefenseCover ID %d", dr.id))

	pressState := &DefensePressState{
		gi:              dr.gi,
		robotId:         dr.id,
		team:            dr.team,
		name:            pressName,
		activityHandler: dr.activityHandler,
		slot:            DefenseSlot{Kind: DefenseRoleCover},
	}
	coverState := &DefensePressState{
		gi:              dr.gi,
		robotId:         dr.id,
		team:            dr.team,
		name:            coverName,
		activityHandler: dr.activityHandler,
		slot:            DefenseSlot{Kind: DefenseRoleCover},
	}

	sm := NewStateMachine(pressState)
	sm.AddTransition(pressName, "ATTACKER_NEAR", coverState)
	sm.AddTransition(coverName, "ATTACKER_FAR", pressState)

	dr.sm = sm
	dr.pressState = pressState
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
