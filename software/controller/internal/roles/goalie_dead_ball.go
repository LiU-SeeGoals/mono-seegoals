package roles

import (
	"math"
	"time"

	coreai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	deadBallStillDuration     = 3 * time.Second
	deadBallMaxTrackedSpeed   = 0.04
	deadBallStillRadius       = 60.0
	deadBallGoalieMaxDistance = 1900.0
	goalieRescueControlRadius = 420.0

	goalieChipKickSpeed         = 2
	goalieSimChipSpeed          = 3.0
	goalieSimChipAngleInDegrees = 45
)

type deadBallTracker struct {
	active bool
	since  time.Time
	anchor info.Position
}

type GoalieCollectDeadBallState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            StateName
	activityHandler *coreai.ActivityHandler
}

type GoalieSafeAlignState struct {
	Gi              *info.GameInfo
	RobotId         info.ID
	Team            info.Team
	Name            StateName
	ActivityHandler *coreai.ActivityHandler
	Ctx             *GoalieSafeClearIntent
}

type GoalieSafeKickState struct {
	Name            StateName
	RobotId         info.ID
	Team            info.Team
	KickAct         *act.KickBall
	Gi              *info.GameInfo
	ActivityHandler *coreai.ActivityHandler
	Ctx             *GoalieSafeClearIntent
}

type GoalieSafeClearIntent struct {
	gi       *info.GameInfo
	team     info.Team
	selfID   info.ID
	fallback info.Position
	target   info.Position
	frozen   bool
	found    bool
}

func (t *deadBallTracker) Observe(gi *info.GameInfo, team info.Team) {
	ballPos, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil || !goalieDeadBallBaseCandidate(gi, team, ballPos) {
		t.Reset()
		return
	}

	if t.active && t.anchor.Dist2d(ballPos) <= deadBallStillRadius {
		return
	}

	t.active = true
	t.since = time.Now()
	t.anchor = ballPos
}

func (t *deadBallTracker) Reset() {
	t.active = false
	t.since = time.Time{}
	t.anchor = info.Position{}
}

func (t *deadBallTracker) StillLongEnough() bool {
	return t.active && time.Since(t.since) >= deadBallStillDuration
}

func (gr *GoalieRole) ShouldCollectDeadBall() bool {
	if gr.HasBallControl(GoalieBallControlRadius) || !gr.deadBall.StillLongEnough() {
		return false
	}

	ballPos, err := gr.gi.State.GetBall().GetEstimatedPosition()
	if err != nil || !goalieDeadBallBaseCandidate(gr.gi, gr.team, ballPos) {
		return false
	}

	goaliePos, err := gr.gi.State.GetRobotPosition(gr.team, gr.id)
	if err != nil || goaliePos.Dist2d(ballPos) > deadBallGoalieMaxDistance {
		return false
	}

	return true
}

func (s *GoalieCollectDeadBallState) Initialize() {}

func (s *GoalieCollectDeadBallState) GetName() StateName {
	return s.name
}

func (s *GoalieCollectDeadBallState) Update() EventName {
	if goalieHasBallControl(s.gi, s.team, s.robotId, goalieRescueControlRadius) {
		return "BALL_OWNER"
	}

	ballPos, err := s.gi.State.GetBall().GetEstimatedPosition()
	if err != nil || !goalieDeadBallBaseCandidate(s.gi, s.team, ballPos) {
		return "BALL_LOST"
	}

	s.activityHandler.AddActivity(act.NewGoalieCollectDeadBall(s.team, s.robotId))
	return "NONE"
}

func (s *GoalieSafeAlignState) Initialize() {
	s.Ctx.FreezeTarget()
}

func (s *GoalieSafeAlignState) GetName() StateName {
	return s.Name
}

func (s *GoalieSafeAlignState) Update() EventName {
	if !goalieHasBallControl(s.Gi, s.Team, s.RobotId, goalieRescueControlRadius) {
		s.Ctx.ResetTarget()
		return "BALL_LOST"
	}

	if !s.Ctx.HasSafeTarget() {
		s.ActivityHandler.AddActivity(act.NewGoalieCollectDeadBall(s.Team, s.RobotId))
		return "NONE"
	}

	activity := act.NewDirectAlign(s.Team, s.RobotId, s.Ctx.GetTargetPosition(), s.Ctx.GetFromPosition())
	activity.AllowGoalArea(true)
	activity.AllowOutsideField(true)
	activity.AllowBehindGoalLine(true)
	s.ActivityHandler.AddActivity(activity)
	if activity.Achieved(s.Gi) {
		return "ALIGNED"
	}
	return "NONE"
}

func (s *GoalieSafeKickState) Initialize() {
	if !s.Ctx.HasSafeTarget() {
		s.KickAct = nil
		return
	}
	s.KickAct = act.NewKickBall(s.Team, s.RobotId, s.Ctx.GetTargetPosition(), s.Ctx.GetFromPosition())
	s.KickAct.AllowGoalArea(true)
	s.KickAct.AllowOutsideField(true)
	s.KickAct.AllowBehindGoalLine(true)
	s.KickAct.SetKickSpeed(goalieChipKickSpeed)
	s.KickAct.SetSimKickSpeed(goalieSimChipSpeed)
	s.KickAct.SetKickAngle(goalieSimChipAngleInDegrees)
	s.ActivityHandler.AddActivity(s.KickAct)
}

func (s *GoalieSafeKickState) GetName() StateName {
	return s.Name
}

func (s *GoalieSafeKickState) Update() EventName {
	if s.KickAct == nil {
		s.Ctx.ResetTarget()
		return "BALL_LOST"
	}
	if s.KickAct.Achieved(s.Gi) {
		s.Ctx.ResetTarget()
		return "KICKED"
	}
	return "NONE"
}

func (gc *GoalieSafeClearIntent) FreezeTarget() {
	if gc.frozen && gc.found {
		return
	}
	target, ok := chooseSimpleGoalieChipTarget(gc.gi, gc.team, gc.selfID)
	if !ok {
		gc.frozen = false
		gc.found = false
		return
	}
	gc.target = target
	gc.frozen = true
	gc.found = true
}

func (gc *GoalieSafeClearIntent) ResetTarget() {
	gc.frozen = false
	gc.found = false
	gc.target = info.Position{}
}

func (gc *GoalieSafeClearIntent) HasSafeTarget() bool {
	if gc.frozen && gc.found {
		return true
	}
	gc.FreezeTarget()
	return gc.found
}

func (gc *GoalieSafeClearIntent) GetTargetPosition() info.Position {
	if gc.HasSafeTarget() {
		return gc.target
	}
	return gc.fallback
}

func (gc *GoalieSafeClearIntent) GetFromPosition() info.Position {
	pos, _ := gc.gi.State.GetBall().GetEstimatedPosition()
	return pos
}

func goalieDeadBallBaseCandidate(gi *info.GameInfo, team info.Team, ballPos info.Position) bool {
	if gi == nil || gi.State == nil || !gi.HasField() || !goalieRescueAllowedByRef(gi) {
		return false
	}
	if gi.State.GetBall().GetPossessor() != nil {
		return false
	}
	if !ballInOwnGoalArea(gi, team, ballPos) {
		return false
	}
	if vel, ok := gi.State.GetTrackedBall().GetTrackedVelocity(); ok && vel.Norm2d() > deadBallMaxTrackedSpeed {
		return false
	}
	return true
}

func goalieRescueAllowedByRef(gi *info.GameInfo) bool {
	if gi.Status == nil || gi.Status.GetGameEvent() == nil {
		return true
	}
	ge := gi.Status.GetGameEvent()
	switch ge.CurrentState {
	case info.STATE_STOPPED, info.STATE_BALL_PLACEMENT, info.STATE_FREE_KICK,
		info.STATE_KICKOFF_PREPARATION, info.STATE_PENALTY_PREPARATION, info.STATE_TIMEOUT:
		return false
	case info.STATE_HALTED:
		return ge.RefCommand == info.UNINITIALIZED
	default:
		return true
	}
}

func goalieHasBallControl(gi *info.GameInfo, team info.Team, id info.ID, radius float64) bool {
	robotPos, err := gi.State.GetRobotPosition(team, id)
	if err != nil {
		return false
	}
	ballPos, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return false
	}
	return robotPos.Dist2d(ballPos) <= radius
}

func ballInOwnGoalArea(gi *info.GameInfo, team info.Team, ballPos info.Position) bool {
	zone, ok := ownGoalNoGoZone(gi, team)
	return ok && zone.contains(ballPos)
}

func ownGoalNoGoZone(gi *info.GameInfo, team info.Team) (interceptGoalNoGoZone, bool) {
	if defenseXSign(gi, team) > 0 {
		return interceptGoalNoGoZoneFromLines(gi, "RightPenaltyStretch", "RightGoalLine")
	}
	return interceptGoalNoGoZoneFromLines(gi, "LeftPenaltyStretch", "LeftGoalLine")
}

func chooseSimpleGoalieChipTarget(gi *info.GameInfo, team info.Team, selfID info.ID) (info.Position, bool) {
	if gi == nil || gi.State == nil || !gi.HasField() {
		return info.Position{}, false
	}

	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	field := gi.FieldSize()
	halfX := field.X / 2
	halfY := field.Y / 2
	if halfX <= 0 {
		halfX = 4500
	}
	if halfY <= 0 {
		halfY = 3000
	}

	xSign := defenseXSign(gi, team)
	fieldMargin := 500.0
	target := info.Position{
		X: -xSign * halfX * 0.25,
		Y: math.Max(-halfY+fieldMargin, math.Min(halfY-fieldMargin, ballPos.Y)),
		Z: 0,
	}
	_ = selfID
	return target, true
}
