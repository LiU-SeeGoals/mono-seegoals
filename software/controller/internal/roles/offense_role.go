package roles

import (
	"fmt"
	"math"

	ai "github.com/LiU-SeeGoals/controller/internal/ai"
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
	receiveTarget   info.Position
	slot            OffenseSlot
}

type OffenseRoleKind string

const (
	OffenseRoleChaser   OffenseRoleKind = "chaser"
	OffenseRoleReceiver OffenseRoleKind = "receiver"
)

type OffenseSlot struct {
	Kind OffenseRoleKind
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

func isGoalShotAvailable(team info.Team, from info.Position, gi *GameInfo) bool {

	// Try to shoot goal if the sight is clear
	// Otherwise pass to a homie

	goalPosition := gi.EnemyGoalCenter(team)

	// check that the goal is in range
	if from.Dist2d(goalPosition) > 200 { // TODO: I have no idea what a reasonable shooting distance is
		return false
	}

	// check that there are no enemies in the way

	enemies := gi.State.GetOtherTeam(team)
	for i := 0; i < int(TEAM_SIZE); i++ {
		enemyPos, err := enemies[i].GetPosition()
		if err != nil {
			continue
		}
		dist := info.DistToLineSegment(goalPosition.ToV2(), from.ToV2(), enemyPos.ToV2())
		if dist < 200 {
			return false
		}
	}

	return true
}

func (kr *AttemptGoalIntent) SetReceivers(receiverIDs []info.ID) {
	kr.receiverIDs = append(kr.receiverIDs[:0], receiverIDs...)
}

func (kr *AttemptGoalIntent) bestReceiverID() (info.ID, bool) {
	bestID := info.ID(0)
	bestScore := math.Inf(-1)
	found := false

	ballPos, _ := kr.gi.State.GetBall().GetEstimatedPosition()
	goal := kr.gi.EnemyGoalCenter(kr.team)

	for _, id := range kr.receiverIDs {
		if id == kr.id {
			continue
		}

		pos, err := kr.gi.State.GetRobotPosition(kr.team, id)
		if err != nil {
			continue
		}

		distFromBall := ballPos.Dist2d(pos)
		progressTowardGoal := -pos.Dist2d(goal)
		passLengthPenalty := 0.2 * distFromBall

		score := progressTowardGoal - passLengthPenalty
		if score > bestScore {
			bestScore = score
			bestID = id
			found = true
		}
	}

	return bestID, found
}

func (kr *AttemptGoalIntent) chooseKickDecision() KickDecision {
	ballPos, _ := kr.gi.State.GetBall().GetEstimatedPosition()
	goalPosition := kr.gi.EnemyGoalCenter(kr.team)
	robotPos, err := kr.gi.State.GetRobotPosition(kr.team, kr.id)

	// Try to shoot goal if the sight is clear
	if err != nil {
		fmt.Println("failed robot pos")
	}

	if isGoalShotAvailable(kr.team, robotPos, kr.gi) {
		return KickDecision{
			Target: goalPosition,
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
}

type supportOffset struct {
	forward float64
	lateral float64
}

var supportOffsets = []supportOffset{
	{forward: 9990, lateral: 0},
	{forward: 9990, lateral: 0},
	{forward: 9990, lateral: 0},
	{forward: 9990, lateral: 0},
	{forward: 9990, lateral: 0},
	{forward: 9990, lateral: 0},
}

func clamp(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

func (kr *SupportAttackIntent) supportCandidate(ballPos, goalPos info.Position, offset supportOffset) info.Position {
	field := kr.gi.FieldSize()
	margin := 400.0
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

	x := ballPos.X + forwardX*offset.forward + lateralX*offset.lateral
	y := ballPos.Y + forwardY*offset.forward + lateralY*offset.lateral
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

	// Some smarter way of selecting tactical positions should be done
	// here in order to help the main offensive player
	currentPos, err := kr.gi.State.GetRobotPosition(kr.team, kr.id)
	if err != nil {
		return info.Position{}
	}

	ballPos, err := kr.gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return currentPos
	}

	goalPos := kr.gi.EnemyGoalCenter(kr.team)
	slot := int(kr.id) % len(supportOffsets)
	fallback := kr.supportCandidate(ballPos, goalPos, supportOffsets[slot])

	for i := 0; i < len(supportOffsets); i++ {
		offset := supportOffsets[(slot+i)%len(supportOffsets)]
		candidate := kr.supportCandidate(ballPos, goalPos, offset)
		if isGoalShotAvailable(kr.team, candidate, kr.gi) {
			return candidate
		}
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
	supportContext := SupportAttackIntent{gi: kr.gi, team: kr.team, id: kr.id}
	kr.intent = offenseContext

	awaitBall := &SupportState{ctx: &supportContext, gi: kr.gi, team: kr.team, robotId: kr.id, name: awaitName, activityHandler: kr.activityHandler}
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
			"OffenseRole %d BALL_LOST: slot=%s state=%s -> %s\n",
			kr.id,
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
}

func (kr *OffenseRole) SetPassReceivers(receiverIDs []info.ID) {
	if kr.intent == nil {
		return
	}
	kr.intent.SetReceivers(receiverIDs)
}
