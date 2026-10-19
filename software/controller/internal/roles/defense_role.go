package roles

import (
	"fmt"
	"math"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const coverYClamp = 1500.0

const (
	forwardDefenderBallOffset  = 700.0
	middleDefenderBallOffset   = 1700.0
	backwardDefenderDepth      = 3200.0
	highDefenderBallOffset     = 900.0
	defenderMinimumDepth       = 3000.0
	CoverDangerousGoalDistance = 2500.0
	CoverDangerousBallDistance = 500.0
	CoverReleaseGoalDistance   = 2700.0
	CoverReleaseBallDistance   = 350.0
	coverReceiverWeight        = 4.0 / 5.0
	defenseKickBallDistance    = 600.0
	defenseKickMaxBallSpeed    = 0.3
	defenseShooterBallDist     = 350.0
)

var neutralHoldPos = info.Position{X: 0, Y: -1500, Z: 0, Angle: 0}
var defenseKickTarget = info.Position{X: 0, Y: 0, Z: 0, Angle: 0}

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

type coverOpponentSelection struct {
	id       info.ID
	assigned bool
	managed  bool
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
	coverOpponent   *coverOpponentSelection
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
		target = LowDefenderTarget(s.gi, s.team, ballPos, s.slot.LateralOffset)
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

// LowDefenderTarget is shared by role assignment and the running defense role,
// so the robot selected for Low Def is evaluated against the same position it
// will actually drive to. The +/-3000 line is a minimum defensive depth; once
// the ball moves deeper into our half, Low Def may retreat with it.
func LowDefenderTarget(gi *info.GameInfo, team info.Team, ballPos info.Position, lateralOffset float64) info.Position {
	xSign := defenseXSign(gi, team)
	targetDepth := math.Max(defenderMinimumDepth, xSign*ballPos.X)
	targetX := xSign * targetDepth
	targetY := ballPos.Y
	state := DefensePressState{gi: gi, team: team}
	if aimedY, ok := state.opponentAimYAtX(ballPos, targetX); ok {
		targetY = aimedY
	} else if gi != nil && gi.State != nil {
		if projected, ok := gi.State.ProjectBallTrajectoryToX(targetX); ok {
			targetY = projected.Y
		}
	}

	target := info.Position{
		X: targetX,
		Y: targetY + lateralOffset,
		Z: 0,
	}
	target = clampCoverTarget(target)
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

// opponentAimYAtX returns where the likely shooter's facing direction crosses
// targetX. It is local to the defense role because it is a defensive positioning
// heuristic, not a general ball trajectory.
func (s *DefensePressState) opponentAimYAtX(ballPos info.Position, targetX float64) (float64, bool) {
	shooter, found := s.opponentAimingAtBall(ballPos)
	if !found {
		return 0, false
	}

	shooterPos, err := shooter.GetPosition()
	if err != nil {
		return 0, false
	}
	directionX := math.Cos(shooterPos.Angle)
	if math.Abs(directionX) < 1e-6 {
		return 0, false
	}

	t := (targetX - shooterPos.X) / directionX
	if t < 0 {
		return 0, false
	}
	return shooterPos.Y + t*math.Sin(shooterPos.Angle), true
}

func (s *DefensePressState) opponentAimingAtBall(ballPos info.Position) (*info.Robot, bool) {
	if s.gi == nil || s.gi.State == nil {
		return nil, false
	}

	ball := s.gi.State.GetBall()
	if possessor := ball.GetPossessor(); possessor != nil && possessor.GetTeam() != s.team &&
		possessor.IsActive() && ballPos.Dist2d(possessor.DribblerPos()) <= defenseShooterBallDist {
		return possessor, true
	}

	var shooter *info.Robot
	closestDistance := math.Inf(1)
	for _, opponent := range s.gi.State.GetOtherTeam(s.team) {
		if opponent == nil || !opponent.IsActive() {
			continue
		}
		distance := ballPos.Dist2d(opponent.DribblerPos())
		if distance <= defenseShooterBallDist && distance < closestDistance {
			shooter = opponent
			closestDistance = distance
		}
	}

	return shooter, shooter != nil
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

// coverParkingPos returns where Cover should wait when no dangerous opponent is found.
func (s *DefensePressState) coverParkingPos(ballPos info.Position) info.Position {
	target := info.Position{
		X: defenseXSign(s.gi, s.team) * defenderMinimumDepth,
		Y: ballPos.Y,
		Z: 0,
	}
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

// coverTarget places the Cover defender between the ball and the most dangerous opponent.
func (s *DefensePressState) coverTarget(ballPos info.Position) info.Position {
	_, dangerousPos, found := s.selectedCoverOpponent(ballPos)
	if !found {
		return s.coverParkingPos(ballPos)
	}

	target := info.Position{
		X: ballPos.X + (dangerousPos.X-ballPos.X)*coverReceiverWeight,
		Y: ballPos.Y + (dangerousPos.Y-ballPos.Y)*coverReceiverWeight,
		Z: 0,
	}
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (s *DefensePressState) selectedCoverOpponent(ballPos info.Position) (*info.Robot, info.Position, bool) {
	if s.coverOpponent != nil && s.coverOpponent.managed {
		if !s.coverOpponent.assigned {
			return nil, info.Position{}, false
		}
		return LockedCoverThreat(s.gi, s.team, s.coverOpponent.id, ballPos)
	}

	opponent, position, goalDistance, found := s.closestOpponentToOwnGoal()
	if !found || goalDistance > CoverDangerousGoalDistance {
		return nil, info.Position{}, false
	}
	if position.Dist2d(ballPos) <= CoverDangerousBallDistance {
		return nil, info.Position{}, false
	}
	return opponent, position, true
}

// CoverThreat returns an active opponent only while it qualifies as a pass
// threat: close enough to our goal and sufficiently far from the ball carrier.
func CoverThreat(
	gi *info.GameInfo,
	team info.Team,
	opponentID info.ID,
	ballPos info.Position,
) (*info.Robot, info.Position, bool) {
	return coverThreatWithin(
		gi,
		team,
		opponentID,
		ballPos,
		CoverDangerousGoalDistance,
		CoverDangerousBallDistance,
	)
}

// LockedCoverThreat uses wider release limits than CoverThreat. This
// hysteresis prevents vision jitter around a threshold from repeatedly
// releasing and reacquiring the same opponent.
func LockedCoverThreat(
	gi *info.GameInfo,
	team info.Team,
	opponentID info.ID,
	ballPos info.Position,
) (*info.Robot, info.Position, bool) {
	return coverThreatWithin(
		gi,
		team,
		opponentID,
		ballPos,
		CoverReleaseGoalDistance,
		CoverReleaseBallDistance,
	)
}

func coverThreatWithin(
	gi *info.GameInfo,
	team info.Team,
	opponentID info.ID,
	ballPos info.Position,
	maxGoalDistance float64,
	minBallDistance float64,
) (*info.Robot, info.Position, bool) {
	if gi == nil || gi.State == nil {
		return nil, info.Position{}, false
	}
	opponents := gi.State.GetOtherTeam(team)
	if int(opponentID) >= len(opponents) {
		return nil, info.Position{}, false
	}
	opponent := opponents[opponentID]
	if opponent == nil || !opponent.IsActive() {
		return nil, info.Position{}, false
	}
	opponentPos, err := opponent.GetPosition()
	if err != nil {
		return nil, info.Position{}, false
	}

	goalPos := ownGoalCenter(gi, team)
	if opponentPos.Dist2d(goalPos) > maxGoalDistance {
		return nil, info.Position{}, false
	}
	if opponentPos.Dist2d(ballPos) <= minBallDistance {
		return nil, info.Position{}, false
	}
	return opponent, opponentPos, true
}

func ownGoalCenter(gi *info.GameInfo, team info.Team) info.Position {
	halfFieldLength := 4500.0
	if gi != nil && gi.HasField() {
		fieldSize := gi.FieldSize()
		if fieldSize.X > 0 {
			halfFieldLength = fieldSize.X / 2
		}
	}
	return info.Position{X: gi.OwnHalfXSign(team) * halfFieldLength}
}

// closestOpponentToOwnGoal finds the most dangerous opponent by distance to our goal.
func (s *DefensePressState) closestOpponentToOwnGoal() (*info.Robot, info.Position, float64, bool) {
	if s.gi == nil || s.gi.State == nil {
		return nil, info.Position{}, 0, false
	}

	goalPos := ownGoalCenter(s.gi, s.team)

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

		distance := opponentPos.Dist2d(goalPos)
		if distance < closestDistance {
			closestOpponent = opponent
			closestOpponentPos = opponentPos
			closestDistance = distance
		}
	}

	return closestOpponent, closestOpponentPos, closestDistance, closestOpponent != nil
}

type defenseKickIntent struct {
	gi     *info.GameInfo
	target info.Position
}

func (i *defenseKickIntent) GetTargetPosition() info.Position {
	return i.target
}

func (i *defenseKickIntent) GetFromPosition() info.Position {
	if i.gi == nil || i.gi.State == nil {
		return info.Position{}
	}
	position, _ := i.gi.State.GetBall().GetEstimatedPosition()
	return position
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
	mayPlayBall     bool
	wallPosition    info.Position
	pressState      *DefensePressState
	kickIntent      *defenseKickIntent
	coverOpponent   *coverOpponentSelection
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
	alignName := StateName(fmt.Sprintf("DefenseAlignKick ID %d", dr.id))
	kickName := StateName(fmt.Sprintf("DefenseKick ID %d", dr.id))

	coverOpponent := &coverOpponentSelection{}
	pressState := &DefensePressState{
		gi:              dr.gi,
		robotId:         dr.id,
		team:            dr.team,
		name:            pressName,
		activityHandler: dr.activityHandler,
		slot:            DefenseSlot{Kind: DefenseRoleCover},
		coverOpponent:   coverOpponent,
	}
	coverState := &DefensePressState{
		gi:              dr.gi,
		robotId:         dr.id,
		team:            dr.team,
		name:            coverName,
		activityHandler: dr.activityHandler,
		slot:            DefenseSlot{Kind: DefenseRoleCover},
		coverOpponent:   coverOpponent,
	}
	kickIntent := &defenseKickIntent{gi: dr.gi, target: defenseKickTarget}
	alignState := &AlignState{
		Ctx:             kickIntent,
		Gi:              dr.gi,
		Team:            dr.team,
		RobotId:         dr.id,
		Name:            alignName,
		ActivityHandler: dr.activityHandler,
	}
	kickState := &KickState{
		Ctx:             kickIntent,
		Name:            kickName,
		Gi:              dr.gi,
		Team:            dr.team,
		RobotId:         dr.id,
		ActivityHandler: dr.activityHandler,
	}

	sm := NewStateMachine(pressState)
	sm.AddTransition(pressName, "ATTACKER_NEAR", coverState)
	sm.AddTransition(coverName, "ATTACKER_FAR", pressState)
	sm.AddTransition(pressName, "BALL_SHOOTABLE", alignState)
	sm.AddTransition(coverName, "BALL_SHOOTABLE", alignState)
	sm.AddTransition(alignName, "ALIGNED", kickState)
	sm.AddTransition(alignName, "BALL_LOST", pressState)
	sm.AddTransition(kickName, "KICKED", pressState)
	sm.AddTransition(kickName, "BALL_LOST", pressState)

	dr.sm = sm
	dr.pressState = pressState
	dr.kickIntent = kickIntent
	dr.coverOpponent = coverOpponent
}

func (dr *DefenseRole) Run() {
	if dr.defenderBallReadyToKick() {
		dr.sm.TriggerEvent("BALL_SHOOTABLE")
	} else {
		dr.sm.TriggerEvent("BALL_LOST")
	}
	dr.sm.Update()
}

// SetMayPlayBall is controlled by the plan. Only the one defender selected as
// the team's ball actor may align with and kick the ball.
func (dr *DefenseRole) SetMayPlayBall(mayPlay bool) {
	dr.mayPlayBall = mayPlay
}

// SetCoverOpponent supplies the shared plan's marking decision. Calling it
// with assigned=false keeps this role under plan control but parks it until a
// unique dangerous opponent becomes available.
func (dr *DefenseRole) SetCoverOpponent(opponentID info.ID, assigned bool) {
	if dr == nil || dr.coverOpponent == nil {
		return
	}
	dr.coverOpponent.id = opponentID
	dr.coverOpponent.assigned = assigned
	dr.coverOpponent.managed = true
}

// CanKickBall reports whether this defender is physically able to clear the
// ball. It deliberately does not include the plan's ball-actor assignment.
func (dr *DefenseRole) CanKickBall() bool {
	if dr == nil || dr.gi == nil || dr.gi.State == nil || dr.pressState == nil {
		return false
	}
	roleKind := dr.pressState.slot.Kind
	if roleKind != DefenseRoleCover && roleKind != DefenseRoleLow {
		return false
	}

	robotPos, err := dr.gi.State.GetRobotPosition(dr.team, dr.id)
	if err != nil {
		return false
	}
	ballPos, err := dr.gi.State.GetBall().GetEstimatedPosition()
	if err != nil || robotPos.Dist2d(ballPos) > defenseKickBallDistance {
		return false
	}
	ballVelocity, ok := dr.gi.State.GetTrackedBall().GetFreshTrackedVelocity(time.Now(), info.TrackedBallFreshness)
	return ok && ballVelocity.Norm2d() <= defenseKickMaxBallSpeed
}

// Returns whether the plan-selected Cover or Low Def may kick, based on ball
// speed and distance.
func (dr *DefenseRole) defenderBallReadyToKick() bool {
	return dr != nil && dr.mayPlayBall && dr.CanKickBall()
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
