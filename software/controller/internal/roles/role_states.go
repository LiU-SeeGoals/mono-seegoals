package roles

import (
	"math"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/ai/pathplanner"
	sm "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const interceptNoGoWaitClearance = pathplanner.MotionRadius

const alignTransitionConfirmTime = 0 * time.Millisecond

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
	Name            sm.StateName
	ActivityHandler *ai.ActivityHandler
	Ctx             TargetContext
	alignedSince    time.Time
}

func (s *AlignState) Initialize() {
	s.alignedSince = time.Time{}
	if ctx, ok := s.Ctx.(FreezableTargetContext); ok {
		ctx.FreezeTarget()
	}
}

func (s *AlignState) GetName() sm.StateName {
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
func (s *AlignState) Update() sm.EventName {

	targetPos := s.Ctx.GetTargetPosition()
	fromPos := s.Ctx.GetFromPosition()

	var activity act.Activity

	if enemyCloseToBall(s.Gi, s.Team, fromPos, 1000) {
		activity = act.NewDirectAlign(s.Team, s.RobotId, targetPos, fromPos)
	} else {
		activity = act.NewAlign(s.Team, s.RobotId, targetPos, fromPos)
	}
	//activity := act.NewAlign(s.Team, s.RobotId, s.Ctx.GetTargetPosition(), s.Ctx.GetFromPosition())
	s.ActivityHandler.AddActivity(activity)
	if updateAlignConfirmation(&s.alignedSince, activity.Achieved(s.Gi), time.Now()) {
		return "ALIGNED"
	}
	return "NONE"
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
	name            sm.StateName
	activityHandler *ai.ActivityHandler
	ctx             TargetContext
}

func (s *SupportState) Initialize() {
	if ctx, ok := s.ctx.(FreezableTargetContext); ok {
		ctx.ResetTarget()
	}
}

func (s *SupportState) GetName() sm.StateName {
	return s.name
}

func (s *SupportState) Update() sm.EventName {

	activity := act.NewMoveToPosition(s.team, s.robotId, s.ctx.GetFromPosition())
	s.activityHandler.AddActivity(activity)
	achieved := activity.Achieved(s.gi)
	if achieved {
		return "WAITING"
	}
	return "NONE"
}

type ReceivePassState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            sm.StateName
	target          *info.Position
	activityHandler *ai.ActivityHandler
}

func (s *ReceivePassState) Initialize() {
}

func (s *ReceivePassState) GetName() sm.StateName {
	return s.name
}

func (s *ReceivePassState) Update() sm.EventName {
	target := *s.target
	ballPos, err := s.gi.State.GetBall().GetEstimatedPosition()
	if err == nil {
		target.Angle = target.AngleToPosition(ballPos)
	}

	activity := act.NewMoveToPosition(s.team, s.robotId, target)
	activity.SetUseRRT(false)
	activity.AvoidBall(false)
	activity.SetDribble(true)
	s.activityHandler.AddActivity(activity)

	possessor := s.gi.State.GetBall().GetPossessor()
	if possessor != nil && possessor.GetTeam() == s.team && possessor.GetID() == s.robotId {
		return "BALL_RECEIVED"
	}

	return "NONE"
}

type InterceptBallState struct {
	gi              *info.GameInfo
	robotId         info.ID
	team            info.Team
	name            sm.StateName
	activityHandler *ai.ActivityHandler
	ctx             TargetContext
}

func (s *InterceptBallState) Initialize() {
}

func (s *InterceptBallState) GetName() sm.StateName {
	return s.name
}

func (s *InterceptBallState) Update() sm.EventName {
	if ballPos, err := s.gi.State.GetBall().GetEstimatedPosition(); err == nil {
		if waitPos, ok := interceptWaitPositionOutsideGoalNoGo(s.gi, ballPos); ok {
			waitPos.Angle = waitPos.AngleToPosition(ballPos)
			activity := act.NewMoveToPosition(s.team, s.robotId, waitPos)
			s.activityHandler.AddActivity(activity)
			return "NONE"
		}
	}

	ballVel, ok := s.gi.State.GetTrackedBall().GetTrackedVelocity()
	if ok && ballVel.Norm2d() > 0.3 {
		activity := act.NewMoveToBall(s.team, s.robotId)
		s.activityHandler.AddActivity(activity)

		if activity.Achieved(s.gi) {
			return "BALL_OWNER"
		}

		return "NONE"
	}

	targetPos := s.ctx.GetTargetPosition()
	fromPos := s.ctx.GetFromPosition()
	activity := act.NewAlign(s.team, s.robotId, targetPos, fromPos)
	s.activityHandler.AddActivity(activity)

	if activity.Achieved(s.gi) {
		return "BALL_OWNER"
	}

	return "NONE"
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
	if zone, ok := interceptGoalNoGoZoneFromLines(gi, "LeftPenaltyStretch", "LeftGoalLine"); ok {
		zones = append(zones, zone)
	}
	if zone, ok := interceptGoalNoGoZoneFromLines(gi, "RightPenaltyStretch", "RightGoalLine"); ok {
		zones = append(zones, zone)
	}
	return zones
}

func interceptGoalNoGoZoneFromLines(gi *info.GameInfo, frontLineName, backLineName string) (interceptGoalNoGoZone, bool) {
	front := gi.GetFieldLine(frontLineName)
	back := gi.GetFieldLine(backLineName)
	if front == nil || back == nil || front.GetP1() == nil || front.GetP2() == nil || back.GetP1() == nil {
		return interceptGoalNoGoZone{}, false
	}

	frontX := float64(front.GetP1().GetX())
	backX := float64(back.GetP1().GetX())
	y1 := float64(front.GetP1().GetY())
	y2 := float64(front.GetP2().GetY())
	margin := pathplanner.GoalLineSafetyRadius

	return interceptGoalNoGoZone{
		minX: math.Min(frontX, backX) - margin,
		maxX: math.Max(frontX, backX) + margin,
		minY: math.Min(y1, y2) - margin,
		maxY: math.Max(y1, y2) + margin,
	}, true
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
	Name            sm.StateName
	RobotId         info.ID
	Team            info.Team
	KickAct         *act.KickBall
	Gi              *info.GameInfo
	ActivityHandler *ai.ActivityHandler
	Ctx             TargetContext
}

func (s *KickState) Initialize() {

	s.KickAct = act.NewKickBall(s.Team, s.RobotId, s.Ctx.GetTargetPosition(), s.Ctx.GetFromPosition())
	s.ActivityHandler.AddActivity(s.KickAct)
}

func (s *KickState) GetName() sm.StateName {
	return s.Name
}

func (s *KickState) Update() sm.EventName {
	if s.KickAct.Achieved(s.Gi) {
		if ctx, ok := s.Ctx.(FreezableTargetContext); ok {
			ctx.ResetTarget()
		}
		return "KICKED"
	}
	return "NONE"
}
