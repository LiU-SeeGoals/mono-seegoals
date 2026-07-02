package roles

import (
	"fmt"
	"math"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/ai/pathplanner"
	. "github.com/LiU-SeeGoals/controller/internal/frameworks/state_machine"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
)

type OffenseRole struct {
	id              info.ID
	sm              *StateMachine
	activityHandler *ai.ActivityHandler
	gi              *GameInfo
	team            Team
	intent          *AttemptGoalIntent
	support         *SupportAttackIntent
	receiveTarget   info.Position
	slot            OffenseSlot
}

type OffenseRoleKind string

const (
	OffenseRoleChaser   OffenseRoleKind = "chaser"
	OffenseRoleReceiver OffenseRoleKind = "receiver"
	OffenseRoleShooter  OffenseRoleKind = "shooter"
)

type OffenseSlot struct {
	Kind  OffenseRoleKind
	Index int
	Count int
}

type PassTarget struct {
	ReceiverID info.ID
	Position   info.Position
}
type KickDecision struct {
	Target     info.Position
	From       info.Position
	ReceiverID info.ID
	IsPass     bool
}

func NewOffenseRole(robotID ID, activityHandler ai.ActivityHandler, gi *GameInfo, team Team) *OffenseRole {
	return &OffenseRole{
		id:              robotID,
		sm:              nil,
		activityHandler: &activityHandler,
		gi:              gi,
		team:            team,
	}
}

type AttemptGoalIntent struct {
	gi          *GameInfo
	team        Team
	id          ID
	frozen      bool
	decision    KickDecision
	receiverIDs []info.ID
}

const (
	minGoalShotSamples = 3
	maxGoalShotSamples = 9

	// Keep support robots out of the primary attacker's sight line to the goal.
	// This is intentionally wider than the robot body so positioning noise does
	// not leave a support robot clipping the shot lane.
	supportGoalSightClearance = 400.0
)

func goalShotBlockRadius() float64 {
	return pathplanner.MotionRadius + info.BallRadius
}

func goalMouthTargets(team info.Team, gi *GameInfo) []info.Position {
	if gi == nil || !gi.HasField() {
		return nil
	}

	goalLine := gi.EnemyGoalLine(team)
	if len(goalLine) < 2 {
		return []info.Position{gi.EnemyGoalCenter(team)}
	}

	start := goalLine[0]
	end := goalLine[1]
	goalWidth := start.Dist2d(end)
	if goalWidth < 1 {
		return []info.Position{gi.EnemyGoalCenter(team)}
	}

	samples := int(math.Ceil(goalWidth / (goalShotBlockRadius() * 2)))
	samples = int(clamp(float64(samples), minGoalShotSamples, maxGoalShotSamples))
	if samples%2 == 0 {
		samples++
	}

	delta := end.Sub(&start)
	innerMargin := math.Min(0.2, goalShotBlockRadius()/goalWidth)
	maxOffset := 0.5 - innerMargin
	step := 0.0
	if samples > 1 {
		step = maxOffset / float64(samples/2)
	}

	targets := []info.Position{}
	for offset := 0; offset <= samples/2; offset++ {
		if offset == 0 {
			scaled := delta.Scale(0.5)
			targets = append(targets, start.Add(&scaled))
			continue
		}

		highRatio := 0.5 + float64(offset)*step
		lowRatio := 0.5 - float64(offset)*step
		high := delta.Scale(highRatio)
		low := delta.Scale(lowRatio)
		targets = append(targets, start.Add(&high))
		targets = append(targets, start.Add(&low))
	}

	return targets
}

func closestRobotDistanceToShot(team info.Team, shooterID info.ID, from, target info.Position, gi *GameInfo) float64 {
	minDist := math.Inf(1)
	checkRobots := func(robots *info.RobotTeam, skipShooter bool) {
		if robots == nil {
			return
		}

		for id, robot := range robots {
			if robot == nil {
				continue
			}
			if skipShooter && info.ID(id) == shooterID {
				continue
			}

			robotPos, err := robot.GetPosition()
			if err != nil {
				continue
			}

			dist := info.DistToLineSegment(target.ToV2(), from.ToV2(), robotPos.ToV2())
			if dist < minDist {
				minDist = dist
			}
		}
	}

	checkRobots(gi.State.GetTeam(team), true)
	checkRobots(gi.State.GetOtherTeam(team), false)
	return minDist
}

func openGoalShotTarget(team info.Team, shooterID info.ID, from info.Position, gi *GameInfo) (info.Position, bool) {
	targets := goalMouthTargets(team, gi)
	if len(targets) == 0 {
		return info.Position{}, false
	}

	for _, target := range targets {
		if closestRobotDistanceToShot(team, shooterID, from, target, gi) >= goalShotBlockRadius() {
			return target, true
		}
	}

	return info.Position{}, false
}

func isGoalShotAvailable(team info.Team, shooterID info.ID, from info.Position, gi *GameInfo) bool {
	_, ok := openGoalShotTarget(team, shooterID, from, gi)
	return ok
}

func (kr *AttemptGoalIntent) SetReceivers(receiverIDs []info.ID) {
	kr.receiverIDs = append(kr.receiverIDs[:0], receiverIDs...)
}

func (kr *AttemptGoalIntent) bestReceiverID() (info.ID, bool) {
	bestID := info.ID(0)
	bestScore := math.Inf(-1)
	found := false

	ballPos, err := kr.gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return bestID, false
	}

	goal := kr.gi.EnemyGoalCenter(kr.team)
	toGoalX := goal.X - ballPos.X
	toGoalY := goal.Y - ballPos.Y
	toGoalDist := math.Sqrt(toGoalX*toGoalX + toGoalY*toGoalY)
	if toGoalDist < 1 {
		return bestID, false
	}

	forwardX := toGoalX / toGoalDist
	forwardY := toGoalY / toGoalDist
	lateralX := -forwardY
	lateralY := forwardX
	field := kr.gi.FieldSize()

	for _, id := range kr.receiverIDs {
		if id == kr.id {
			continue
		}

		pos, err := kr.gi.State.GetRobotPosition(kr.team, id)
		if err != nil {
			continue
		}

		toReceiverX := pos.X - ballPos.X
		toReceiverY := pos.Y - ballPos.Y
		distFromBall := ballPos.Dist2d(pos)
		progressTowardGoal := toReceiverX*forwardX + toReceiverY*forwardY
		lateralMiss := math.Abs(toReceiverX*lateralX + toReceiverY*lateralY)
		passLengthPenalty := 0.15 * distFromBall
		lateralPenalty := 0.1 * lateralMiss

		score := progressTowardGoal - passLengthPenalty - lateralPenalty
		if isGoalShotAvailable(kr.team, id, pos, kr.gi) {
			score += field.X * 0.25
		}
		if score > bestScore {
			bestScore = score
			bestID = id
			found = true
		}
	}

	return bestID, found
}

func (kr *AttemptGoalIntent) chooseKickDecision() KickDecision {
	ballPos, err := kr.gi.State.GetBall().GetEstimatedPosition()
	goalPosition := kr.gi.EnemyGoalCenter(kr.team)
	if err != nil {
		return KickDecision{
			Target: goalPosition,
			From:   goalPosition,
			IsPass: false,
		}
	}

	if shotTarget, ok := openGoalShotTarget(kr.team, kr.id, ballPos, kr.gi); ok {
		return KickDecision{
			Target: shotTarget,
			From:   ballPos,
			IsPass: false,
		}
	}

	receiverID, ok := kr.bestReceiverID()
	if !ok {
		return KickDecision{
			Target: goalPosition,
			From:   ballPos,
			IsPass: false,
		}
	}

	receiverPos, err := kr.gi.State.GetRobotPosition(kr.team, receiverID)
	if err != nil {
		return KickDecision{
			Target: goalPosition,
			From:   ballPos,
			IsPass: false,
		}
	}

	return KickDecision{
		Target:     receiverPos,
		From:       ballPos,
		ReceiverID: receiverID,
		IsPass:     true,
	}
}

func (kr *AttemptGoalIntent) FreezeTarget() {
	if kr.frozen {
		return
	}

	kr.decision = kr.chooseKickDecision()
	kr.frozen = true
}

func (kr *AttemptGoalIntent) ResetTarget() {
	kr.frozen = false
	kr.decision = KickDecision{}
}

func (kr *AttemptGoalIntent) CurrentDecision() KickDecision {
	if !kr.frozen {
		return kr.chooseKickDecision()
	}

	return kr.decision
}

func (kr *AttemptGoalIntent) GetTargetPosition() info.Position {
	return kr.CurrentDecision().Target
}

func (kr *AttemptGoalIntent) GetFromPosition() info.Position {
	return kr.CurrentDecision().From
}

type SupportAttackIntent struct {
	gi   *GameInfo
	team Team
	id   ID
	slot OffenseSlot
}

func (kr *SupportAttackIntent) SetSlot(slot OffenseSlot) {
	kr.slot = slot
}

func clamp(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

func normalizedSupportSlot(slot OffenseSlot) OffenseSlot {
	if slot.Count <= 0 {
		slot.Count = 1
	}
	if slot.Index < 0 {
		slot.Index = 0
	}
	if slot.Index >= slot.Count {
		slot.Index = slot.Count - 1
	}
	return slot
}

func supportLane(slot OffenseSlot) float64 {
	slot = normalizedSupportSlot(slot)
	return float64(slot.Index) - float64(slot.Count-1)/2.0
}

func supportLaneOptions(slot OffenseSlot, id ID) []float64 {
	slot = normalizedSupportSlot(slot)
	lane := supportLane(slot)
	if slot.Count == 1 {
		if int(id)%2 == 0 {
			return []float64{-0.5, 0.5, 0}
		}
		return []float64{0.5, -0.5, 0}
	}
	if lane == 0 {
		return []float64{0, -0.5, 0.5}
	}
	return []float64{lane, lane * 0.5, -lane}
}

func supportDepthOptions(slot OffenseSlot, lane float64) []float64 {
	slot = normalizedSupportSlot(slot)
	maxLane := math.Max(1, float64(slot.Count-1)/2.0)
	centerBias := 1 - math.Min(1, math.Abs(lane)/maxLane)
	base := 0.58 + 0.15*centerBias
	return []float64{
		base,
		math.Min(0.85, base+0.12),
		math.Max(0.40, base-0.12),
	}
}

func supportPositionBlocksGoalSight(ballPos, goalPos, supportPos info.Position) bool {
	distance := info.DistToLineSegment(goalPos.ToV2(), ballPos.ToV2(), supportPos.ToV2())
	return distance < supportGoalSightClearance
}

func (kr *SupportAttackIntent) supportCandidate(
	ballPos, goalPos info.Position,
	forwardFraction float64,
	lane float64,
) info.Position {
	field := kr.gi.FieldSize()
	margin := pathplanner.RobotSafetyRadius
	halfX := field.X/2 - margin
	halfY := field.Y/2 - margin

	toGoalX := goalPos.X - ballPos.X
	toGoalY := goalPos.Y - ballPos.Y
	dist := math.Sqrt(toGoalX*toGoalX + toGoalY*toGoalY)
	if dist < 1 {
		pos, _ := kr.gi.State.GetRobotPosition(kr.team, kr.id)
		return pos
	}

	forwardX := toGoalX / dist
	forwardY := toGoalY / dist
	lateralX := -forwardY
	lateralY := forwardX

	slot := normalizedSupportSlot(kr.slot)
	lateralSpacing := field.Y / float64(slot.Count+2)
	maxLateral := field.Y * 0.35
	lateralOffset := clamp(lane*lateralSpacing, -maxLateral, maxLateral)
	forwardOffset := dist * forwardFraction
	x := ballPos.X + forwardX*forwardOffset + lateralX*lateralOffset
	y := ballPos.Y + forwardY*forwardOffset + lateralY*lateralOffset
	target := info.Position{
		X: x,
		Y: y,
		Z: 0,
	}
	target.X = clamp(target.X, -halfX, halfX)
	target.Y = clamp(target.Y, -halfY, halfY)
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (kr *SupportAttackIntent) GetFromPosition() info.Position {
	currentPos, err := kr.gi.State.GetRobotPosition(kr.team, kr.id)
	if err != nil {
		return info.Position{}
	}

	ballPos, err := kr.gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return currentPos
	}

	goalPos := kr.gi.EnemyGoalCenter(kr.team)
	laneOptions := supportLaneOptions(kr.slot, kr.id)
	fallbackLane := supportLane(kr.slot)
	if len(laneOptions) > 0 {
		fallbackLane = laneOptions[0]
	}
	fallback := kr.supportCandidate(ballPos, goalPos, supportDepthOptions(kr.slot, fallbackLane)[0], fallbackLane)
	best := fallback
	bestScore := math.Inf(-1)
	foundClearShot := false
	bestSafeFallback := fallback
	bestSafeFallbackScore := math.Inf(-1)
	foundSafeFallback := false

	for _, lane := range laneOptions {
		for _, depth := range supportDepthOptions(kr.slot, lane) {
			candidate := kr.supportCandidate(ballPos, goalPos, depth, lane)
			if supportPositionBlocksGoalSight(ballPos, goalPos, candidate) {
				continue
			}

			score := -candidate.Dist2d(goalPos)
			if score > bestSafeFallbackScore {
				bestSafeFallbackScore = score
				bestSafeFallback = candidate
				foundSafeFallback = true
			}
			if !isGoalShotAvailable(kr.team, kr.id, candidate, kr.gi) {
				continue
			}

			if score > bestScore {
				bestScore = score
				best = candidate
				foundClearShot = true
			}
		}
	}

	if foundClearShot {
		return best
	}
	if foundSafeFallback {
		return bestSafeFallback
	}
	return fallback
}

func (kr *SupportAttackIntent) GetTargetPosition() info.Position {
	pos, _ := kr.gi.State.GetBall().GetEstimatedPosition()

	return pos
}

func (kr *OffenseRole) Init() {
	awaitName := StateName(fmt.Sprintf("Support ID %d", kr.id))
	kickPrepareName := StateName(fmt.Sprintf("KickPrepare ID %d", kr.id))
	kickName := StateName(fmt.Sprintf("Kick ID %d", kr.id))
	receiveName := StateName(fmt.Sprintf("ReceivePass ID %d", kr.id))
	interceptName := StateName(fmt.Sprintf("InterceptBall ID %d", kr.id))

	offenseContext := &AttemptGoalIntent{gi: kr.gi, team: kr.team, id: kr.id}
	supportContext := &SupportAttackIntent{gi: kr.gi, team: kr.team, id: kr.id}
	kr.intent = offenseContext
	kr.support = supportContext

	awaitBall := &SupportState{ctx: supportContext, gi: kr.gi, team: kr.team, robotId: kr.id, name: awaitName, activityHandler: kr.activityHandler}
	prepareKick := &AlignState{Ctx: offenseContext, Gi: kr.gi, Team: kr.team, RobotId: kr.id, Name: kickPrepareName, ActivityHandler: kr.activityHandler}
	kick := &KickState{Ctx: offenseContext, Name: kickName, Gi: kr.gi, Team: kr.team, RobotId: kr.id, ActivityHandler: kr.activityHandler}
	intercept := &InterceptBallState{
		gi:              kr.gi,
		robotId:         kr.id,
		team:            kr.team,
		name:            interceptName,
		activityHandler: kr.activityHandler,
		ctx:             offenseContext,
	}
	receivePass := &ReceivePassState{
		gi:              kr.gi,
		robotId:         kr.id,
		team:            kr.team,
		name:            receiveName,
		target:          &kr.receiveTarget,
		activityHandler: kr.activityHandler,
	}

	sm := NewStateMachine(awaitBall)

	sm.AddTransition(awaitName, "BALL_OWNER", prepareKick)
	sm.AddTransition(awaitName, "BALL_APPROACHING", intercept)
	sm.AddTransition(awaitName, "PASS_TARGETED", receivePass)

	sm.AddTransition(interceptName, "BALL_OWNER", prepareKick)
	sm.AddTransition(interceptName, "BALL_LOST", awaitBall)

	sm.AddTransition(kickPrepareName, "ALIGNED", kick)
	sm.AddTransition(kickPrepareName, "BALL_LOST", awaitBall)
	sm.AddTransition(kickName, "KICKED", prepareKick)
	sm.AddTransition(kickName, "BALL_LOST", awaitBall)
	sm.AddTransition(receiveName, "BALL_OWNER", prepareKick)
	sm.AddTransition(receiveName, "BALL_RECEIVED", prepareKick)
	sm.AddTransition(receiveName, "BALL_APPROACHING", intercept)
	sm.AddTransition(receiveName, "BALL_LOST", awaitBall)

	kr.sm = sm
	kr.slot = OffenseSlot{Kind: OffenseRoleChaser}
}

func (kr *OffenseRole) Run() {
	kr.sm.Update()
}

func (kr *OffenseRole) TriggerEvent(event EventName) {
	stateName := kr.sm.CurrentStateName()
	transitions := kr.sm.StateTransitions[stateName]
	nextState, canTransition := transitions[event]
	if event == "BALL_LOST" && canTransition {
		fmt.Printf(
			"OffenseRole %d %s BALL_LOST: slot=%s state=%s -> %s\n",
			kr.id,
			kr.team,
			kr.slot.Kind,
			stateName,
			nextState.GetName(),
		)
		if kr.intent != nil {
			kr.intent.ResetTarget()
		}
	}

	kr.sm.TriggerEvent(event)
}

func (kr *OffenseRole) CurrentDecision() KickDecision {
	if kr.intent == nil {
		return KickDecision{}
	}

	return kr.intent.CurrentDecision()
}

func (kr *OffenseRole) ReceivePass(target info.Position) {
	kr.receiveTarget = target
	kr.sm.TriggerEvent("PASS_TARGETED")
}

func (kr *OffenseRole) SetSlot(slot OffenseSlot) {
	kr.slot = slot
	if kr.support != nil {
		kr.support.SetSlot(slot)
	}
}

func (kr *OffenseRole) SetPassReceivers(receiverIDs []info.ID) {
	if kr.intent == nil {
		return
	}
	kr.intent.SetReceivers(receiverIDs)
}
