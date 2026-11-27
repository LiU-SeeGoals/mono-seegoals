package roles

import (
	"github.com/LiU-SeeGoals/controller/internal/rules"
	"math"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/ai/pathplanner"
	"github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const interceptNoGoWaitClearance = pathplanner.MotionRadius

const alignTransitionConfirmTime = 0 * time.Millisecond

const (
	directAlignEnemyRadius   = 1000.0
	directAlignReleaseRadius = 1200.0
)

type TargetContext interface {
	GetTargetPosition() info.Position
	GetFromPosition() info.Position
}

type FreezableTargetContext interface {
	FreezeTarget()
	ResetTarget()
}

type AlignState struct {
	Gi              *info.GameInfo
	RobotId         info.ID
	Team            info.Team
	Name            state_machine.StateName
	ActivityHandler *ai.ActivityHandler
	Ctx             TargetContext
	alignedSince    time.Time
	// Keep obstacle-aware staging for restarts even when a defender is nearby.
	PlanApproach bool
	latch        *activity.AlignLatch
}

func (s *AlignState) Initialize() {
	s.alignedSince = time.Time{}
	s.latch = &activity.AlignLatch{}
	if ctx, ok := s.Ctx.(FreezableTargetContext); ok {
		ctx.FreezeTarget()
	}
}

func (s *AlignState) GetName() state_machine.StateName {
	return s.Name
}
func enemyCloseToBall(gi *info.GameInfo, team info.Team, ballPos info.Position, radius float64) bool {
	enemies := gi.State.GetOtherTeam(team)

	for _, enemy := range enemies {
		enemyPos, err := enemy.GetPosition()

		if err != nil {
			continue
		}

		if ballPos.Dist2d(enemyPos) < radius {
			return true
		}
	}

	return false
}
func (s *AlignState) Update() state_machine.EventName {

	targetPos := s.Ctx.GetTargetPosition()
	fromPos := s.Ctx.GetFromPosition()
	if s.latch == nil {
		s.latch = &activity.AlignLatch{}
	}

	var act *activity.AlignBall

	direct := !s.PlanApproach && s.latch.DirectApproach(
		enemyCloseToBall(s.Gi, s.Team, fromPos, directAlignEnemyRadius),
		enemyCloseToBall(s.Gi, s.Team, fromPos, directAlignReleaseRadius),
		s.Gi.Now(),
	)
	if direct {
		act = activity.NewDirectAlign(s.Team, s.RobotId, targetPos, fromPos)
	} else {
		act = activity.NewAlign(s.Team, s.RobotId, targetPos, fromPos)
	}
	act.SetLatch(s.latch)
	//activity := act.NewAlign(s.Team, s.RobotId, s.Ctx.GetTargetPosition(), s.Ctx.GetFromPosition())
	s.ActivityHandler.AddActivity(act)
	if updateAlignConfirmation(&s.alignedSince, act.Achieved(s.Gi), s.Gi.Now()) {
		return EventAligned
	}
	return state_machine.NoEvent
}

func updateAlignConfirmation(alignedSince *time.Time, aligned bool, now time.Time) bool {
	// Require consecutive aligned frames so a fast orbit correction cannot
	// complete merely by crossing the lateral threshold once.
	if !aligned {
		*alignedSince = time.Time{}
		return false
	}
	if alignedSince.IsZero() {
		*alignedSince = now
		return false
	}
	return now.Sub(*alignedSince) >= alignTransitionConfirmTime
}

type SupportState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            state_machine.StateName
	activityHandler *ai.ActivityHandler
	ctx             TargetContext
}

func (s *SupportState) Initialize() {
	if ctx, ok := s.ctx.(FreezableTargetContext); ok {
		ctx.ResetTarget()
	}
}

func (s *SupportState) GetName() state_machine.StateName {
	return s.name
}

func (s *SupportState) Update() state_machine.EventName {

	s.activityHandler.AddActivity(activity.NewMoveToPosition(s.team, s.robotId, s.ctx.GetFromPosition()))
	return state_machine.NoEvent
}

type ReceivePassState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            state_machine.StateName
	target          *info.Position
	activityHandler *ai.ActivityHandler
}

func (s *ReceivePassState) Initialize() {
}

func (s *ReceivePassState) GetName() state_machine.StateName {
	return s.name
}

func (s *ReceivePassState) Update() state_machine.EventName {
	target := *s.target
	ballPos, err := s.gi.State.GetBall().GetEstimatedPosition()
	if err == nil {
		target.Angle = target.AngleToPosition(ballPos)
	}

	act := activity.NewMoveToPosition(s.team, s.robotId, target)
	act.SetUseRRT(false)
	act.AvoidBall(false)
	act.SetDribble(true)
	s.activityHandler.AddActivity(act)

	possessor := s.gi.State.GetBall().GetPossessor()
	if possessor != nil && possessor.GetTeam() == s.team && possessor.GetID() == s.robotId {
		return EventBallReceived
	}

	return state_machine.NoEvent
}

type InterceptBallState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            state_machine.StateName
	activityHandler *ai.ActivityHandler
	ctx             TargetContext
}

func (s *InterceptBallState) Initialize() {
}

func (s *InterceptBallState) GetName() state_machine.StateName {
	return s.name
}

func (s *InterceptBallState) Update() state_machine.EventName {

	ballPos, posErr := s.gi.State.GetBall().GetEstimatedPosition()

	ballState, ok := s.gi.State.GetTrackedBall().GetFreshBallState(s.gi.Now(), 200*time.Millisecond)

	ballMoving := false

	if ok {
		ballPos = ballState.Position
		ballMoving = ballState.Velocity.Norm2d() > 0.3
	}

	if ok || posErr == nil {
		if waitPos, ok := interceptWaitPositionOutsideGoalNoGo(s.gi, ballPos); ok {
			waitPos.Angle = waitPos.AngleToPosition(ballPos)
			act := activity.NewMoveToPosition(s.team, s.robotId, waitPos)
			s.activityHandler.AddActivity(act)
			return state_machine.NoEvent
		}
	}

	if ballMoving {
		act := activity.NewMoveToBall(s.team, s.robotId)
		s.activityHandler.AddActivity(act)

		if act.Achieved(s.gi) {
			return EventBallOwner
		}

		return state_machine.NoEvent
	}

	targetPos := s.ctx.GetTargetPosition()
	fromPos := s.ctx.GetFromPosition()
	act := activity.NewAlign(s.team, s.robotId, targetPos, fromPos)
	s.activityHandler.AddActivity(act)

	if act.Achieved(s.gi) {
		return EventBallOwner
	}

	return state_machine.NoEvent
}

type interceptGoalNoGoZone struct {
	minX float64
	maxX float64
	minY float64
	maxY float64
}

func interceptWaitPositionOutsideGoalNoGo(gi *info.GameInfo, ballPos info.Position) (info.Position, bool) {
	for _, zone := range interceptGoalNoGoZones(gi) {
		if !zone.contains(ballPos) {
			continue
		}
		return zone.waitPosition(ballPos), true
	}

	return info.Position{}, false
}

func interceptGoalNoGoZones(gi *info.GameInfo) []interceptGoalNoGoZone {
	if gi == nil || !gi.HasField() {
		return nil
	}

	zones := make([]interceptGoalNoGoZone, 0, 2)
	if zone, ok := interceptGoalNoGoZoneForSide(gi, false); ok {
		zones = append(zones, zone)
	}
	if zone, ok := interceptGoalNoGoZoneForSide(gi, true); ok {
		zones = append(zones, zone)
	}
	return zones
}

func interceptGoalNoGoZoneForSide(gi *info.GameInfo, positive bool) (interceptGoalNoGoZone, bool) {
	if gi == nil {
		return interceptGoalNoGoZone{}, false
	}
	for _, area := range gi.DefenseAreas() {
		if (area.FrontX > 0) == positive {
			margin := rules.DefenseAreaMotionClearance
			return interceptGoalNoGoZone{
				minX: math.Min(area.FrontX, area.BackX) - margin,
				maxX: math.Max(area.FrontX, area.BackX) + margin,
				minY: area.MinY - margin, maxY: area.MaxY + margin,
			}, true
		}
	}
	return interceptGoalNoGoZone{}, false
}

func (z interceptGoalNoGoZone) contains(pos info.Position) bool {
	return pos.X >= z.minX && pos.X <= z.maxX &&
		pos.Y >= z.minY && pos.Y <= z.maxY
}

func (z interceptGoalNoGoZone) waitPosition(ballPos info.Position) info.Position {
	waitPos := ballPos
	if z.maxX <= 0 {
		waitPos.X = z.maxX + interceptNoGoWaitClearance
	} else {
		waitPos.X = z.minX - interceptNoGoWaitClearance
	}
	waitPos.Y = math.Max(z.minY, math.Min(z.maxY, waitPos.Y))
	return waitPos
}

type KickState struct {
	Name            state_machine.StateName
	RobotId         info.ID
	Team            info.Team
	KickAct         *activity.KickBall
	Gi              *info.GameInfo
	ActivityHandler *ai.ActivityHandler
	Ctx             TargetContext
}

func (s *KickState) Initialize() {

	s.KickAct = activity.NewKickBall(s.Team, s.RobotId, s.Ctx.GetTargetPosition(), s.Ctx.GetFromPosition())
	s.ActivityHandler.AddActivity(s.KickAct)
}

func (s *KickState) GetName() state_machine.StateName {
	return s.Name
}

func (s *KickState) Update() state_machine.EventName {
	if s.KickAct.Achieved(s.Gi) {
		if ctx, ok := s.Ctx.(FreezableTargetContext); ok {
			ctx.ResetTarget()
		}
		return EventKicked
	}
	if s.KickAct.NeedsRealign(s.Gi) {
		return EventMisaligned
	}
	return state_machine.NoEvent
}
