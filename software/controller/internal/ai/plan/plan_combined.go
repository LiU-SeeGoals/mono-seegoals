package plan

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/debugstate"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/referee"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

// CombinedPlan is the open-play planner: it elects a goalie, assigns the
// field robots to tactical slots for the current mode (attack/defend) and
// steps their roles once per frame. The slot table lives in
// slot_assignment.go, role bookkeeping in role_manager.go and defender/goalie
// positioning in defense_wall.go.
type CombinedPlan struct {
	plannerCore
	ballTouchRestrictedRobot atomic.Uint32

	// Match state below is reset by Init and built on the first Tick.
	initialized bool
	// gi is this frame's GameInfo. Roles keep a pointer to it, so Tick
	// overwrites it in place instead of replacing it.
	gi                info.GameInfo
	roleManager       *combinedRoleManager
	possessionTracker ballPossessionTracker
	actorTracker      offenseBallActorTracker
	frameMonitor      *helper.FrameSkipMonitor

	mode               tacticalMode
	candidateMode      tacticalMode
	candidateModeSince time.Time

	goalieID   info.ID
	goalieRole *roles.GoalieRole

	pass pendingPass
}

// pendingPass is the pass the planner last ordered, until the receiver has
// the ball or the pass is abandoned.
type pendingPass struct {
	active   bool
	receiver info.ID
	start    time.Time
}

type tacticalMode string

const (
	tacticalModeAttack tacticalMode = "attack"
	tacticalModeDefend tacticalMode = "defend"

	tacticalModeSwitchDelay = 500 * time.Millisecond

	passInFlightTimeout    = 2 * time.Second
	ballMovingSpeed        = 0.3 // m/s
	trackedBallVelocityAge = 200 * time.Millisecond
)

func NewCombinedPlan(team info.Team) *CombinedPlan {
	return &CombinedPlan{
		plannerCore: plannerCore{team: team},
	}
}

func (m *CombinedPlan) Init(
	activities *[info.TEAM_SIZE]activity.Activity,
	lock *sync.Mutex,
	team info.Team) {
	m.ActivityHandler.Activities = activities
	m.ActivityHandler.Activity_lock = lock
	m.team = team
	m.Active = true
	m.initialized = false
}

func (m *CombinedPlan) reset() {
	m.gi = info.GameInfo{}
	m.roleManager = newCombinedRoleManager(&m.ActivityHandler, &m.gi, m.team)
	m.possessionTracker = ballPossessionTracker{}
	m.actorTracker = offenseBallActorTracker{}
	m.frameMonitor = helper.NewFrameSkipMonitor(m.team.String() + " combined_plan")
	m.mode = tacticalModeAttack
	m.candidateMode = ""
	m.candidateModeSince = time.Time{}
	m.goalieID = 0
	m.goalieRole = nil
	m.pass = pendingPass{}
	m.initialized = true
}

func (m *CombinedPlan) Tick(gi *info.GameInfo) {
	if !m.Active {
		return
	}
	if !m.initialized {
		m.reset()
	}
	m.gi = *gi
	m.step()
}

func (m *CombinedPlan) Kill() {
	m.Active = false
}

func (m *CombinedPlan) step() {
	gi := &m.gi
	now := gi.Now()
	m.frameMonitor.Observe(gi.VisionFrame())
	possession := m.possessionTracker.update(gi, now)
	rawOwner := observedBallOwner(gi)
	restrictedRobot, hasRestrictedRobot := m.ballTouchRestriction()

	activeRobots := m.activeRobots(gi)
	goalieID, hasGoalie, fieldRobots := m.splitGoalieFromFieldRobots(gi, activeRobots)
	m.updateGoalieRole(goalieID, hasGoalie)
	m.updateMode(possession.owner, now)

	rm := m.roleManager
	slots := m.assignTacticalSlots(
		gi,
		fieldRobots,
		roleAssignmentForMode(len(fieldRobots), m.mode),
		possession.owner,
		restrictedRobot,
		hasRestrictedRobot,
	)
	rm.applySlotAssignments(slots, now)
	if hasRestrictedRobot {
		if restrictedAttacker, ok := rm.attackers[restrictedRobot]; ok {
			restrictedAttacker.TriggerEvent(roles.EventBallLost)
		}
	}

	supportIDs := rm.idsForSlot(tacticalSlotSupportShooter)
	rm.configureSupportShooters(supportIDs)
	receiverIDs := supportIDs
	if hasRestrictedRobot {
		receiverIDs = withoutRobot(receiverIDs, restrictedRobot)
		if m.pass.active && m.pass.receiver == restrictedRobot {
			m.pass.active = false
		}
	}
	rm.configureOffenseReceivers(receiverIDs)

	if isStoppedFrame(gi) {
		m.stepStopped(activeRobots)
		return
	}

	m.stepOffense(possession.owner, rawOwner, restrictedRobot, hasRestrictedRobot)
	m.updateDefenseAndGoaliePositioning(false)
}

// updateGoalieRole (re)creates the goalie role when the elected goalie changes.
func (m *CombinedPlan) updateGoalieRole(goalieID info.ID, hasGoalie bool) {
	if hasGoalie && m.goalieRole != nil && m.goalieID == goalieID {
		return
	}
	if m.goalieRole != nil {
		m.ActivityHandler.ClearActivity(m.goalieID)
		debugstate.ClearRobotRole(m.team, m.goalieID)
		m.goalieRole = nil
	}
	if !hasGoalie {
		return
	}
	m.goalieID = goalieID
	m.goalieRole = roles.NewGoalieRole(goalieID, m.ActivityHandler, m.team, m.goalieClearTarget(&m.gi))
	m.goalieRole.Init()
}

// updateMode follows ball possession with a debounce, except that an
// announced restart during STOP sets the mode immediately.
func (m *CombinedPlan) updateMode(owner trackedBallOwner, now time.Time) {
	if restartMode, announced := stoppedRestartMode(m.gi.Status.GetGameEvent(), m.team); announced {
		m.mode = restartMode
		m.candidateMode = ""
		m.candidateModeSince = time.Time{}
		return
	}
	desired := desiredModeForTrackedOwner(owner, m.team, m.mode)
	m.mode = stableMode(m.mode, desired, &m.candidateMode, &m.candidateModeSince, now)
}

// stepStopped parks the robots during STOP and prepares the announced restart.
func (m *CombinedPlan) stepStopped(activeRobots []info.ID) {
	gi := &m.gi
	rm := m.roleManager
	m.pass.active = false
	m.actorTracker.switchTo(rm.attackers, noOffenseBallActor())
	if referee.PrepareForUpcomingKickoff(gi, m.team, activeRobots, &m.ActivityHandler) {
		return
	}
	for id, attacker := range rm.attackers {
		attacker.TriggerEvent(roles.EventBallLost)
		m.ActivityHandler.AddActivity(activity.NewRefStop(m.team, id))
	}
	m.updateDefenseAndGoaliePositioning(true)
	referee.PrepareKickerForUpcomingFreeKick(gi, m.team, activeRobots, &m.ActivityHandler)
}

// stepOffense picks the one attacker that plays the ball this frame (owner,
// pass receiver or interceptor) and runs every attacker.
func (m *CombinedPlan) stepOffense(
	owner trackedBallOwner,
	rawOwner trackedBallOwner,
	restrictedRobot info.ID,
	hasRestrictedRobot bool,
) {
	gi := &m.gi
	rm := m.roleManager
	offenseRobots := rm.offenseIDs()

	ballVel, ballVelOK := gi.State.GetTrackedBall().GetFreshTrackedVelocity(gi.Now(), trackedBallVelocityAge)
	ballMoving := ballVelOK && ballVel.Norm2d() > ballMovingSpeed
	ownerRetained := owner.valid && ownerStillRetained(gi, owner)

	passInFlight := m.pass.active &&
		containsRobot(offenseRobots, m.pass.receiver) &&
		gi.Now().Sub(m.pass.start) < passInFlightTimeout &&
		(!rawOwner.valid || rawOwner.team != m.team) &&
		(ballMoving || !ownerRetained)
	ownerMayPlayBall := !hasRestrictedRobot || owner.id != restrictedRobot

	handledByOffense := false
	if !passInFlight && owner.valid && owner.team == m.team && ownerMayPlayBall {
		handledByOffense = m.playBallAsOwner(owner.id)
	}

	if !handledByOffense {
		if passInFlight {
			m.actorTracker.switchTo(rm.attackers, newOffenseBallActor(m.pass.receiver, offenseBallActorReceiver))
		} else {
			m.pass.active = false
			m.sendInterceptor(offenseRobots, restrictedRobot, hasRestrictedRobot)
		}
	}

	for _, attacker := range rm.attackers {
		attacker.Run()
	}
}

// playBallAsOwner hands the ball to our owning attacker and, when it decides
// to pass, tells the receiver. It reports false if ownerID has no offense role.
func (m *CombinedPlan) playBallAsOwner(ownerID info.ID) bool {
	rm := m.roleManager
	owner, ok := rm.attackers[ownerID]
	if !ok {
		return false
	}
	m.pass.active = false

	m.actorTracker.switchTo(rm.attackers, newOffenseBallActor(ownerID, offenseBallActorOwner))
	owner.TriggerEvent(roles.EventBallOwner)
	decision := owner.CurrentDecision()
	if decision.IsPass && decision.ReceiverID != ownerID {
		if receiver, ok := rm.attackers[decision.ReceiverID]; ok {
			m.pass = pendingPass{active: true, receiver: decision.ReceiverID, start: m.gi.Now()}
			receiver.ReceivePass(decision.Target)
		}
	}
	return true
}

// sendInterceptor sends the ball chaser (or, without one, the attacker
// closest to the ball) after a loose or opponent ball.
func (m *CombinedPlan) sendInterceptor(offenseRobots []info.ID, restrictedRobot info.ID, hasRestrictedRobot bool) {
	rm := m.roleManager
	interceptorID, hasInterceptor := rm.idForSlotExcluding(tacticalSlotBallChaser, restrictedRobot, hasRestrictedRobot)
	if !hasInterceptor {
		candidates := offenseRobots
		if hasRestrictedRobot {
			candidates = withoutRobot(candidates, restrictedRobot)
		}
		if len(candidates) > 0 {
			interceptorID = m.getRobotForBall(&m.gi, candidates)
			hasInterceptor = true
		}
	}
	if interceptor, ok := rm.attackers[interceptorID]; hasInterceptor && ok {
		m.actorTracker.switchTo(rm.attackers, newOffenseBallActor(interceptorID, offenseBallActorChaser))
		interceptor.TriggerEvent(roles.EventBallApproaching)
		return
	}
	m.actorTracker.switchTo(rm.attackers, noOffenseBallActor())
}

func (m *CombinedPlan) setBallTouchRestriction(id info.ID) {
	// Zero represents no restriction, so robot IDs are stored with an offset.
	m.ballTouchRestrictedRobot.Store(uint32(id) + 1)
}

func (m *CombinedPlan) clearBallTouchRestriction() {
	m.ballTouchRestrictedRobot.Store(0)
}

func (m *CombinedPlan) ballTouchRestriction() (info.ID, bool) {
	encoded := m.ballTouchRestrictedRobot.Load()
	if encoded == 0 {
		return 0, false
	}
	return info.ID(encoded - 1), true
}

func desiredModeForTrackedOwner(owner trackedBallOwner, ourTeam info.Team, fallback tacticalMode) tacticalMode {
	if !owner.valid {
		return fallback
	}
	if owner.team == ourTeam {
		return tacticalModeAttack
	}
	return tacticalModeDefend
}

func stoppedRestartMode(gameEvent *info.GameEvent, ourTeam info.Team) (tacticalMode, bool) {
	if gameEvent == nil || gameEvent.GetCurrentState() != info.STATE_STOPPED {
		return "", false
	}

	restartTeam, announced := gameEvent.NextCommand.FreeKickTeam()
	if !announced {
		restartTeam, announced = gameEvent.NextCommand.KickoffTeam()
	}
	if !announced {
		return "", false
	}
	if restartTeam == ourTeam {
		return tacticalModeAttack, true
	}
	return tacticalModeDefend, true
}

func stableMode(
	current tacticalMode,
	desired tacticalMode,
	candidate *tacticalMode,
	candidateSince *time.Time,
	now time.Time,
) tacticalMode {
	if desired == current {
		*candidate = ""
		*candidateSince = time.Time{}
		return current
	}

	if *candidate != desired {
		*candidate = desired
		*candidateSince = now
		return current
	}

	if now.Sub(*candidateSince) < tacticalModeSwitchDelay {
		return current
	}

	*candidate = ""
	*candidateSince = time.Time{}
	return desired
}
