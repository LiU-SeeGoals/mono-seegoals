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

const (
	forwardDefenderGoalWidthFraction     = 0.7
	middleDefenderGoalWidthFraction      = 0.7
	backwardDefenderGoalWidthFraction    = 0.3
	highDefenderPenaltyDepthFraction     = 0.9
	minimumDefenderGoalWidthFraction     = 0.5
	defenderLateralGoalWidthFraction     = 0.5
	coverDangerousGoalWidthFraction      = 0.5
	coverReleaseGoalWidthFraction        = 0.2
	passReceiverReleaseGoalWidthFraction = 0.3
	CoverDangerousBallDistance           = 500.0
	CoverReleaseBallDistance             = 350.0
	coverReceiverWeight                  = 4.0 / 5.0
	defenseKickBallDistance              = 600.0
	defenseKickMaxBallSpeed              = 0.3
	defenseShooterBallDist               = 350.0
)

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

type defenseTacticalGeometry struct {
	forwardBallOffset       float64
	middleBallOffset        float64
	backwardGoalInset       float64
	highBallOffset          float64
	minimumGoalInset        float64
	lateralLimit            float64
	coverAcquireDistance    float64
	coverReleaseDistance    float64
	passReceiverReleaseDist float64
}

func newDefenseTacticalGeometry(gi *info.GameInfo) (defenseTacticalGeometry, bool) {
	if gi == nil {
		return defenseTacticalGeometry{}, false
	}
	geometry, ok := gi.FieldGeometry()
	if !ok || geometry.Width <= 0 || geometry.GoalWidth <= 0 || geometry.PenaltyAreaDepth <= 0 || geometry.PenaltyAreaWidth <= 0 {
		return defenseTacticalGeometry{}, false
	}
	return defenseTacticalGeometry{
		forwardBallOffset:       geometry.GoalWidth * forwardDefenderGoalWidthFraction,
		middleBallOffset:        geometry.PenaltyAreaDepth + geometry.GoalWidth*middleDefenderGoalWidthFraction,
		backwardGoalInset:       geometry.PenaltyAreaDepth + geometry.GoalWidth*backwardDefenderGoalWidthFraction,
		highBallOffset:          geometry.PenaltyAreaDepth * highDefenderPenaltyDepthFraction,
		minimumGoalInset:        geometry.PenaltyAreaDepth + geometry.GoalWidth*minimumDefenderGoalWidthFraction,
		lateralLimit:            math.Min(geometry.Width/2, geometry.PenaltyAreaWidth/2+geometry.GoalWidth*defenderLateralGoalWidthFraction),
		coverAcquireDistance:    2*geometry.PenaltyAreaDepth + geometry.GoalWidth*coverDangerousGoalWidthFraction,
		coverReleaseDistance:    2*geometry.PenaltyAreaDepth + geometry.GoalWidth*(coverDangerousGoalWidthFraction+coverReleaseGoalWidthFraction),
		passReceiverReleaseDist: geometry.GoalWidth * passReceiverReleaseGoalWidthFraction,
	}, true
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
	wallPosition    *info.Position
}

func (s *DefensePressState) Initialize() {}

func (s *DefensePressState) GetName() StateName {
	return s.name
}

func (s *DefensePressState) Update() EventName {
	if s.gi == nil || s.gi.State == nil {
		return "NONE"
	}
	dimensions, ok := newDefenseTacticalGeometry(s.gi)
	if !ok {
		target, err := s.gi.State.GetRobotPosition(s.team, s.robotId)
		if err == nil {
			s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, target))
		}
		return "NONE"
	}

	ballPos, _ := s.gi.State.GetBall().GetEstimatedPosition()
	var target info.Position
	switch s.slot.Kind {
	case DefenseRoleWall:
		if s.wallPosition != nil {
			target = *s.wallPosition
			target.Angle = target.AngleToPosition(ballPos)
		} else {
			target = s.ballGoalLineTarget(ballPos, dimensions.highBallOffset, s.slot.LateralOffset)
		}
	case DefenseRoleHigh:
		target = s.ballGoalLineTarget(ballPos, dimensions.highBallOffset, s.slot.LateralOffset)
	case DefenseRoleLow:
		target = LowDefenderTarget(s.gi, s.team, ballPos, s.slot.LateralOffset)
	case DefenseRoleForward:
		target = s.ballGoalLineTarget(ballPos, dimensions.forwardBallOffset, s.slot.LateralOffset)
	case DefenseRoleMiddle:
		target = s.ballGoalLineTarget(ballPos, dimensions.middleBallOffset, s.slot.LateralOffset)
	case DefenseRoleBackward:
		target = s.backwardDefenderTarget(ballPos, s.slot.LateralOffset)
	case DefenseRoleCover:
		target = s.coverTarget(ballPos)
	default:
		target = s.backwardDefenderTarget(ballPos, s.slot.LateralOffset)
		target.Angle = target.AngleToPosition(ballPos)
	}

	s.activityHandler.AddActivity(act.NewMoveToPosition(s.team, s.robotId, target))
	return "NONE"
}

func (s *DefensePressState) ballGoalLineTarget(ballPos info.Position, ballOffset float64, lateralOffset float64) info.Position {
	goalPos, ok := s.gi.OwnGoalCenter(s.team)
	if !ok {
		return ballPos
	}
	toGoalX := goalPos.X - ballPos.X
	toGoalY := goalPos.Y - ballPos.Y
	dist := math.Sqrt(toGoalX*toGoalX + toGoalY*toGoalY)
	if dist < 1 {
		hold := goalPos
		hold.X -= defenseXSign(s.gi, s.team) * ballOffset
		hold.Angle = hold.AngleToPosition(ballPos)
		return hold
	}

	target := info.Position{
		X: ballPos.X + toGoalX/dist*ballOffset - toGoalY/dist*lateralOffset,
		Y: ballPos.Y + toGoalY/dist*ballOffset + toGoalX/dist*lateralOffset,
		Z: 0,
	}
	target = clampCoverTarget(s.gi, target)
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (s *DefensePressState) backwardDefenderTarget(ballPos info.Position, lateralOffset float64) info.Position {
	dimensions, ok := newDefenseTacticalGeometry(s.gi)
	if !ok {
		return ballPos
	}
	targetX, ok := defensiveLineX(s.gi, s.team, dimensions.backwardGoalInset)
	if !ok {
		return ballPos
	}
	target := info.Position{
		X: targetX,
		Y: ballPos.Y + lateralOffset,
		Z: 0,
	}
	target = clampCoverTarget(s.gi, target)
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

// LowDefenderTarget is shared by role assignment and the running defense role,
// so the robot selected for Low Def is evaluated against the same position it
// will actually drive to. The minimum line is measured inward from the goal
// line reported by Vision; once the ball moves deeper, Low Def retreats with it.
func LowDefenderTarget(gi *info.GameInfo, team info.Team, ballPos info.Position, lateralOffset float64) info.Position {
	dimensions, ok := newDefenseTacticalGeometry(gi)
	if !ok {
		return ballPos
	}
	minimumX, ok := defensiveLineX(gi, team, dimensions.minimumGoalInset)
	if !ok {
		return ballPos
	}
	xSign := defenseXSign(gi, team)
	targetX := minimumX
	if xSign*ballPos.X > xSign*minimumX {
		targetX = ballPos.X
	}
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
	target = clampCoverTarget(gi, target)
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

func defensiveLineX(gi *info.GameInfo, team info.Team, goalInset float64) (float64, bool) {
	if gi == nil {
		return 0, false
	}
	goalCenter, ok := gi.OwnGoalCenter(team)
	if !ok {
		return 0, false
	}
	return goalCenter.X - gi.OwnHalfXSign(team)*goalInset, true
}

// DefenderLateralLimit returns the shared lateral boundary used by both role
// assignment and the executing defense role.
func DefenderLateralLimit(gi *info.GameInfo) (float64, bool) {
	dimensions, ok := newDefenseTacticalGeometry(gi)
	if !ok {
		return 0, false
	}
	return dimensions.lateralLimit, true
}

func clampCoverTarget(gi *info.GameInfo, target info.Position) info.Position {
	limit, ok := DefenderLateralLimit(gi)
	if !ok {
		return target
	}
	target.Y = math.Max(-limit, math.Min(limit, target.Y))
	return gi.ClampToField(target, 0)
}

// coverParkingPos returns where Cover should wait when no dangerous opponent is found.
func (s *DefensePressState) coverParkingPos(ballPos info.Position) info.Position {
	dimensions, ok := newDefenseTacticalGeometry(s.gi)
	if !ok {
		return ballPos
	}
	targetX, ok := defensiveLineX(s.gi, s.team, dimensions.minimumGoalInset)
	if !ok {
		return ballPos
	}
	target := info.Position{
		X: targetX,
		Y: ballPos.Y,
		Z: 0,
	}
	target = clampCoverTarget(s.gi, target)
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
	target = clampCoverTarget(s.gi, target)
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (s *DefensePressState) selectedCoverOpponent(ballPos info.Position) (*info.Robot, info.Position, bool) {
	if s.coverOpponent != nil && s.coverOpponent.managed {
		if !s.coverOpponent.assigned {
			return nil, info.Position{}, false
		}
		return LockedPassReceiverThreat(s.gi, s.team, s.coverOpponent.id, ballPos)
	}

	opponent, position, goalDistance, found := s.closestOpponentToOwnGoal()
	acquireDistance, _, ok := coverGoalDistances(s.gi)
	if !found || !ok || goalDistance > acquireDistance {
		return nil, info.Position{}, false
	}
	if position.Dist2d(ballPos) <= CoverDangerousBallDistance {
		return nil, info.Position{}, false
	}
	return opponent, position, true
}

// PassReceiverThreat accepts opponents throughout our defensive half. It is
// intentionally wider than CoverThreat so High Def reacts before a receiver
// has already entered the final goal-protection zone.
func PassReceiverThreat(
	gi *info.GameInfo,
	team info.Team,
	opponentID info.ID,
	ballPos info.Position,
) (*info.Robot, info.Position, bool) {
	if gi == nil {
		return nil, info.Position{}, false
	}
	if _, ok := gi.FieldGeometry(); !ok {
		return nil, info.Position{}, false
	}
	return passReceiverThreatWithin(
		gi,
		team,
		opponentID,
		ballPos,
		0,
		CoverDangerousBallDistance,
	)
}

// LockedPassReceiverThreat adds hysteresis to the defensive-half boundary so
// that small vision changes do not release and reacquire the same receiver.
func LockedPassReceiverThreat(
	gi *info.GameInfo,
	team info.Team,
	opponentID info.ID,
	ballPos info.Position,
) (*info.Robot, info.Position, bool) {
	if gi == nil {
		return nil, info.Position{}, false
	}
	if _, ok := gi.FieldGeometry(); !ok {
		return nil, info.Position{}, false
	}
	dimensions, ok := newDefenseTacticalGeometry(gi)
	if !ok {
		return nil, info.Position{}, false
	}
	return passReceiverThreatWithin(
		gi,
		team,
		opponentID,
		ballPos,
		-dimensions.passReceiverReleaseDist,
		CoverReleaseBallDistance,
	)
}

func passReceiverThreatWithin(
	gi *info.GameInfo,
	team info.Team,
	opponentID info.ID,
	ballPos info.Position,
	minimumOwnHalfProgress float64,
	minimumBallDistance float64,
) (*info.Robot, info.Position, bool) {
	opponent, opponentPos, ok := activeOpponent(gi, team, opponentID)
	if !ok {
		return nil, info.Position{}, false
	}
	progressIntoOwnHalf := gi.OwnHalfXSign(team) * opponentPos.X
	if progressIntoOwnHalf < minimumOwnHalfProgress || opponentPos.Dist2d(ballPos) <= minimumBallDistance {
		return nil, info.Position{}, false
	}
	return opponent, opponentPos, true
}

// CoverThreat returns an active opponent only while it qualifies as a pass
// threat: close enough to our goal and sufficiently far from the ball carrier.
func CoverThreat(
	gi *info.GameInfo,
	team info.Team,
	opponentID info.ID,
	ballPos info.Position,
) (*info.Robot, info.Position, bool) {
	acquireDistance, _, ok := coverGoalDistances(gi)
	if !ok {
		return nil, info.Position{}, false
	}
	return coverThreatWithin(
		gi,
		team,
		opponentID,
		ballPos,
		acquireDistance,
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
	_, releaseDistance, ok := coverGoalDistances(gi)
	if !ok {
		return nil, info.Position{}, false
	}
	return coverThreatWithin(
		gi,
		team,
		opponentID,
		ballPos,
		releaseDistance,
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
	opponent, opponentPos, ok := activeOpponent(gi, team, opponentID)
	if !ok {
		return nil, info.Position{}, false
	}

	goalPos, ok := gi.OwnGoalCenter(team)
	if !ok {
		return nil, info.Position{}, false
	}
	if opponentPos.Dist2d(goalPos) > maxGoalDistance {
		return nil, info.Position{}, false
	}
	if opponentPos.Dist2d(ballPos) <= minBallDistance {
		return nil, info.Position{}, false
	}
	return opponent, opponentPos, true
}

func activeOpponent(gi *info.GameInfo, team info.Team, opponentID info.ID) (*info.Robot, info.Position, bool) {
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
	return opponent, opponentPos, true
}

func coverGoalDistances(gi *info.GameInfo) (acquire, release float64, ok bool) {
	dimensions, ok := newDefenseTacticalGeometry(gi)
	if !ok {
		return 0, 0, false
	}
	return dimensions.coverAcquireDistance, dimensions.coverReleaseDistance, true
}

// closestOpponentToOwnGoal finds the most dangerous opponent by distance to our goal.
func (s *DefensePressState) closestOpponentToOwnGoal() (*info.Robot, info.Position, float64, bool) {
	if s.gi == nil || s.gi.State == nil {
		return nil, info.Position{}, 0, false
	}

	goalPos, ok := s.gi.OwnGoalCenter(s.team)
	if !ok {
		return nil, info.Position{}, 0, false
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
		wallPosition:    &dr.wallPosition,
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
