package ai

import (
	"math"
	"sort"
	"sync"
	"time"

	coreai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/debugstate"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

type CombinedPlan struct {
	plannerCore
}

type tacticalMode string

const (
	tacticalModeAttack tacticalMode = "attack"
	tacticalModeDefend tacticalMode = "defend"

	roleSwitchMinDuration    = 750 * time.Millisecond
	tacticalModeSwitchDelay  = 500 * time.Millisecond
	activeRobotTimeoutMillis = int64(1000)

	preferredGoalieID = info.ID(6)

	goalWallDistanceFromGoal = 800.0
	goalWallYClamp           = 1200.0
	goalWallSpacing          = 240.0
	highDefenderBallOffset   = 900.0
	lowDefenderDepth         = 3600.0
)

type roleKind string

const (
	roleOffense roleKind = "offense"
	roleDefense roleKind = "defense"
)

type roleAssignment struct {
	slots []tacticalSlotKind
}

type tacticalSlotKind string

const (
	tacticalSlotBallChaser   tacticalSlotKind = "ball_chaser"
	tacticalSlotBallReceiver tacticalSlotKind = "ball_receiver"
	tacticalSlotDefenderWall tacticalSlotKind = "defender_wall"
	tacticalSlotDefenderHigh tacticalSlotKind = "defender_high"
	tacticalSlotDefenderLow  tacticalSlotKind = "defender_low"
	tacticalSlotGoalie       tacticalSlotKind = "goalie"
)

func (slot tacticalSlotKind) roleKind() roleKind {
	switch slot {
	case tacticalSlotBallChaser, tacticalSlotBallReceiver:
		return roleOffense
	case tacticalSlotDefenderWall, tacticalSlotDefenderHigh, tacticalSlotDefenderLow:
		return roleDefense
	default:
		return ""
	}
}

func (slot tacticalSlotKind) label() string {
	switch slot {
	case tacticalSlotBallChaser:
		return "Chaser"
	case tacticalSlotBallReceiver:
		return "Receiver"
	case tacticalSlotDefenderWall:
		return "Wall"
	case tacticalSlotDefenderHigh:
		return "High Def"
	case tacticalSlotDefenderLow:
		return "Low Def"
	case tacticalSlotGoalie:
		return "Goalie"
	default:
		return ""
	}
}

type combinedRoleManager struct {
	activityHandler *coreai.ActivityHandler
	gi              *GameInfo
	team            Team
	attackers       map[info.ID]*roles.OffenseRole
	defenders       map[info.ID]*roles.DefenseRole
	roleByRobot     map[info.ID]roleKind
	slotByRobot     map[info.ID]tacticalSlotKind
	lastChanged     map[info.ID]time.Time
}

func newCombinedRoleManager(activityHandler *coreai.ActivityHandler, gi *GameInfo, team Team) *combinedRoleManager {
	return &combinedRoleManager{
		activityHandler: activityHandler,
		gi:              gi,
		team:            team,
		attackers:       make(map[info.ID]*roles.OffenseRole),
		defenders:       make(map[info.ID]*roles.DefenseRole),
		roleByRobot:     make(map[info.ID]roleKind),
		slotByRobot:     make(map[info.ID]tacticalSlotKind),
		lastChanged:     make(map[info.ID]time.Time),
	}
}

func (rm *combinedRoleManager) applySlotAssignments(assignments map[info.ID]tacticalSlotKind, now time.Time) {
	desired := make(map[info.ID]roleKind)
	for id, slot := range assignments {
		role := slot.roleKind()
		if role == "" {
			continue
		}
		desired[id] = role
	}

	for id := range rm.roleByRobot {
		if _, ok := desired[id]; !ok {
			rm.remove(id)
		}
	}

	for id, role := range desired {
		current, exists := rm.roleByRobot[id]
		currentSlot, hasSlot := rm.slotByRobot[id]
		nextSlot := assignments[id]
		if exists && current == role && hasSlot && currentSlot == nextSlot {
			continue
		}
		if exists && currentSlot != nextSlot && now.Sub(rm.lastChanged[id]) < roleSwitchMinDuration {
			continue
		}

		if !exists || current != role {
			switch role {
			case roleOffense:
				rm.assignOffense(id, now)
			case roleDefense:
				rm.assignDefense(id, now)
			}
		}

		rm.setSlot(id, nextSlot, now)
	}
}

func (rm *combinedRoleManager) assignOffense(id info.ID, now time.Time) {
	rm.remove(id)
	attacker := roles.NewOffenseRole(id, *rm.activityHandler, rm.gi, rm.team)
	attacker.Init()
	rm.attackers[id] = attacker
	rm.roleByRobot[id] = roleOffense
	rm.lastChanged[id] = now
}

func (rm *combinedRoleManager) assignDefense(id info.ID, now time.Time) {
	rm.remove(id)
	defender := roles.NewDefenseRole(id, *rm.activityHandler, rm.gi, rm.team)
	defender.Init()
	rm.defenders[id] = defender
	rm.roleByRobot[id] = roleDefense
	rm.lastChanged[id] = now
}

func (rm *combinedRoleManager) remove(id info.ID) {
	delete(rm.attackers, id)
	delete(rm.defenders, id)
	delete(rm.roleByRobot, id)
	delete(rm.slotByRobot, id)
	rm.activityHandler.ClearActivity(id)
	debugstate.ClearRobotRole(rm.team, id)
}

func (rm *combinedRoleManager) setSlot(id info.ID, slot tacticalSlotKind, now time.Time) {
	rm.slotByRobot[id] = slot
	rm.lastChanged[id] = now
	debugstate.SetRobotRole(rm.team, id, slot.label())

	switch slot {
	case tacticalSlotBallChaser:
		if attacker, ok := rm.attackers[id]; ok {
			attacker.SetSlot(roles.OffenseSlot{Kind: roles.OffenseRoleChaser})
		}
	case tacticalSlotBallReceiver:
		if attacker, ok := rm.attackers[id]; ok {
			attacker.SetSlot(roles.OffenseSlot{Kind: roles.OffenseRoleReceiver})
		}
	case tacticalSlotDefenderWall:
		if defender, ok := rm.defenders[id]; ok {
			defender.SetSlot(roles.DefenseSlot{Kind: roles.DefenseRoleWall})
		}
	case tacticalSlotDefenderHigh:
		if defender, ok := rm.defenders[id]; ok {
			defender.SetSlot(roles.DefenseSlot{Kind: roles.DefenseRoleHigh})
		}
	case tacticalSlotDefenderLow:
		if defender, ok := rm.defenders[id]; ok {
			defender.SetSlot(roles.DefenseSlot{Kind: roles.DefenseRoleLow})
		}
	}
}

func (rm *combinedRoleManager) offenseIDs() []info.ID {
	ids := make([]info.ID, 0, len(rm.attackers))
	for id := range rm.attackers {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func (rm *combinedRoleManager) defenseIDs() []info.ID {
	ids := make([]info.ID, 0, len(rm.defenders))
	for id := range rm.defenders {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func (rm *combinedRoleManager) idsForSlot(slot tacticalSlotKind) []info.ID {
	ids := []info.ID{}
	for id, assignedSlot := range rm.slotByRobot {
		if assignedSlot == slot {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func (rm *combinedRoleManager) idForSlot(slot tacticalSlotKind) (info.ID, bool) {
	ids := rm.idsForSlot(slot)
	if len(ids) == 0 {
		return 0, false
	}
	return ids[0], true
}

func (rm *combinedRoleManager) configureOffenseReceivers(receiverIDs []info.ID) {
	for _, attacker := range rm.attackers {
		attacker.SetPassReceivers(receiverIDs)
	}
}

func NewCombinedPlan(team Team) *CombinedPlan {
	return &CombinedPlan{
		plannerCore: plannerCore{team: team},
	}
}

func (m *CombinedPlan) Init(
	incoming <-chan GameInfo,
	activities *[TEAM_SIZE]act.Activity,
	lock *sync.Mutex,
	team Team,
) {
	m.incomingGameInfo = incoming
	m.ActivityHandler.Activities = activities
	m.ActivityHandler.Activity_lock = lock
	m.team = team
	m.Active = true

	go m.run()
}

func (m *CombinedPlan) getRobotClosestToPosition(
	gi *info.GameInfo,
	activeRobots []info.ID,
	target info.Position,
) info.ID {
	if len(activeRobots) == 0 {
		return info.ID(0)
	}

	closestId := activeRobots[0]
	bestDist := math.Inf(1)

	for _, id := range activeRobots {
		robotPos, err := gi.State.GetRobotPosition(m.team, id)
		if err != nil {
			continue
		}

		dist := robotPos.Dist2d(target)
		if dist < bestDist {
			bestDist = dist
			closestId = id
		}
	}

	return closestId
}

func (m *CombinedPlan) getRobotForBall(gi *info.GameInfo, activeRobots []info.ID) info.ID {
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	ballVel, ok := gi.State.GetTrackedBall().GetTrackedVelocity()

	targetPos := ballPos

	if ok && ballVel.Norm2d() > 0.3 {
		lookaheadSeconds := 0.5

		targetPos.X = ballPos.X + ballVel.X*1000*lookaheadSeconds
		targetPos.Y = ballPos.Y + ballVel.Y*1000*lookaheadSeconds
	}

	return m.getRobotClosestToPosition(gi, activeRobots, targetPos)
}

func (m *CombinedPlan) activeRobots(gi *GameInfo) []info.ID {
	team := gi.State.GetTeam(m.team)
	nowMillis := time.Now().UnixMilli()
	recent := []info.ID{}
	seen := []info.ID{}

	for id, robot := range team {
		if robot == nil || !robot.IsActive() {
			continue
		}

		_, robotTime, err := robot.GetPositionTime()
		if err != nil {
			continue
		}

		robotID := info.ID(id)
		seen = append(seen, robotID)
		if nowMillis-robotTime <= activeRobotTimeoutMillis {
			recent = append(recent, robotID)
		}
	}

	if len(recent) > 0 {
		return recent
	}
	return seen
}

func containsRobot(robots []info.ID, target info.ID) bool {
	for _, id := range robots {
		if id == target {
			return true
		}
	}
	return false
}

func withoutRobot(robots []info.ID, excluded info.ID) []info.ID {
	filtered := []info.ID{}
	for _, id := range robots {
		if id != excluded {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

func (m *CombinedPlan) chooseGoalie(gi *GameInfo, activeRobots []info.ID) (info.ID, bool) {
	if containsRobot(activeRobots, preferredGoalieID) {
		return preferredGoalieID, true
	}
	if len(activeRobots) == 0 {
		return 0, false
	}

	return m.getRobotClosestToPosition(gi, activeRobots, m.defendedGoalCenter(gi)), true
}

func (m *CombinedPlan) desiredMode(gi *GameInfo, fallback tacticalMode) tacticalMode {
	possessor := gi.State.GetBall().GetPossessor()
	if possessor == nil {
		return fallback
	}
	if possessor.GetTeam() == m.team {
		return tacticalModeAttack
	}
	return tacticalModeDefend
}

func desiredModeForTrackedOwner(owner trackedBallOwner, ourTeam Team, fallback tacticalMode) tacticalMode {
	if !owner.valid {
		return fallback
	}
	if owner.team == ourTeam {
		return tacticalModeAttack
	}
	return tacticalModeDefend
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

func roleAssignmentForMode(total int, mode tacticalMode) roleAssignment {
	if mode == tacticalModeDefend {
		switch total {
		case 0:
			return roleAssignment{}
		case 1:
			return roleAssignment{slots: []tacticalSlotKind{tacticalSlotBallChaser}}
		case 2:
			return roleAssignment{slots: []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotBallReceiver}}
		case 3:
			return roleAssignment{slots: []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotDefenderWall, tacticalSlotDefenderLow}}
		case 4:
			return roleAssignment{slots: []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotDefenderWall, tacticalSlotDefenderLow, tacticalSlotDefenderHigh}}
		default:
			return roleAssignment{slots: []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotBallReceiver, tacticalSlotDefenderWall, tacticalSlotDefenderLow, tacticalSlotDefenderHigh}}
		}
	}

	switch total {
	case 0:
		return roleAssignment{}
	case 1:
		return roleAssignment{slots: []tacticalSlotKind{tacticalSlotBallChaser}}
	case 2:
		return roleAssignment{slots: []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotBallReceiver}}
	case 3:
		return roleAssignment{slots: []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotBallReceiver, tacticalSlotDefenderLow}}
	case 4:
		return roleAssignment{slots: []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotBallReceiver, tacticalSlotDefenderLow, tacticalSlotDefenderHigh}}
	default:
		return roleAssignment{slots: []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotBallReceiver, tacticalSlotDefenderLow, tacticalSlotDefenderHigh, tacticalSlotDefenderWall}}
	}
}

func clampFloat(value float64, minValue float64, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

func (m *CombinedPlan) chooseBallChaser(gi *GameInfo, candidates []info.ID, owner trackedBallOwner) info.ID {
	if owner.valid && owner.team == m.team && containsRobot(candidates, owner.id) {
		return owner.id
	}

	possessor := gi.State.GetBall().GetPossessor()
	if possessor != nil && possessor.GetTeam() == m.team && containsRobot(candidates, possessor.GetID()) {
		return possessor.GetID()
	}
	return m.getRobotForBall(gi, candidates)
}

func (m *CombinedPlan) chooseReceiver(gi *GameInfo, candidates []info.ID) info.ID {
	if len(candidates) == 0 {
		return 0
	}

	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	goal := gi.EnemyGoalCenter(m.team)
	bestID := candidates[0]
	bestScore := math.Inf(-1)

	for _, id := range candidates {
		pos, err := gi.State.GetRobotPosition(m.team, id)
		if err != nil {
			continue
		}

		progressTowardGoal := -pos.Dist2d(goal)
		passLengthPenalty := 0.2 * pos.Dist2d(ballPos)
		score := progressTowardGoal - passLengthPenalty
		if score > bestScore {
			bestScore = score
			bestID = id
		}
	}

	return bestID
}

func (m *CombinedPlan) highDefenderTarget(gi *GameInfo, ballPos Position) Position {
	goalPos := m.defendedGoalCenter(gi)
	toGoal := goalPos.Sub(&ballPos)
	dist := toGoal.Norm2d()
	if dist < 1 {
		target := Position{X: -m.defenseXSign(gi) * 1000, Y: 0, Z: 0, Angle: 0}
		target.Angle = target.AngleToPosition(ballPos)
		return target
	}

	step := toGoal.Normalize2d().Scale(highDefenderBallOffset)
	target := ballPos.Add(&step)
	target.Y = clampFloat(target.Y, -goalWallYClamp, goalWallYClamp)
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (m *CombinedPlan) lowDefenderTarget(gi *GameInfo, ballPos Position) Position {
	target := Position{
		X: m.defenseXSign(gi) * lowDefenderDepth,
		Y: clampFloat(ballPos.Y, -goalWallYClamp, goalWallYClamp),
		Z: 0,
	}
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (m *CombinedPlan) targetForTacticalSlot(gi *GameInfo, slot tacticalSlotKind) Position {
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()

	switch slot {
	case tacticalSlotDefenderWall:
		wallPositions := m.calcGoalWallPositionsForRobots(gi, ballPos, 1)
		if len(wallPositions) > 0 {
			return wallPositions[0]
		}
	case tacticalSlotDefenderHigh:
		return m.highDefenderTarget(gi, ballPos)
	case tacticalSlotDefenderLow:
		return m.lowDefenderTarget(gi, ballPos)
	}

	return ballPos
}

func (m *CombinedPlan) assignTacticalSlots(
	gi *GameInfo,
	robots []info.ID,
	assignment roleAssignment,
	owner trackedBallOwner,
) map[info.ID]tacticalSlotKind {
	assignments := make(map[info.ID]tacticalSlotKind)
	available := append([]info.ID{}, robots...)

	for _, slot := range assignment.slots {
		if len(available) == 0 {
			break
		}

		var id info.ID
		switch slot {
		case tacticalSlotBallChaser:
			id = m.chooseBallChaser(gi, available, owner)
		case tacticalSlotBallReceiver:
			id = m.chooseReceiver(gi, available)
		default:
			id = m.getRobotClosestToPosition(gi, available, m.targetForTacticalSlot(gi, slot))
		}

		assignments[id] = slot
		available = withoutRobot(available, id)
	}

	return assignments
}

func (m *CombinedPlan) getAttackerPosition(gi *GameInfo) (Position, bool) {
	opponents := gi.State.GetOtherTeam(m.team)
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()

	minDist := math.Inf(1)
	var closestPos Position
	found := false

	var i ID
	for i = 0; i < TEAM_SIZE; i++ {
		robotPos, err := opponents[i].GetPosition()
		if err != nil {
			continue
		}
		dist := ballPos.Dist2d(robotPos)
		if dist < minDist {
			minDist = dist
			closestPos = robotPos
			found = true
		}
	}
	return closestPos, found
}

func (m *CombinedPlan) defendsPositiveHalf(gi *GameInfo) bool {
	isBlueTeam := m.team == Blue
	isBlueOnPositiveHalf := gi.Status.GetBlueTeamOnPositiveHalf()
	return (isBlueTeam && isBlueOnPositiveHalf) || (!isBlueTeam && !isBlueOnPositiveHalf)
}

func (m *CombinedPlan) defenseXSign(gi *GameInfo) float64 {
	if m.defendsPositiveHalf(gi) {
		return 1.0
	}
	return -1.0
}

func (m *CombinedPlan) defendedGoalCenter(gi *GameInfo) Position {
	goalCenter := blueGoalCenter
	goalCenter.X = m.defenseXSign(gi) * math.Abs(blueGoalCenter.X)
	return goalCenter
}

func (m *CombinedPlan) attackerIsThreatening(gi *GameInfo, attackerPos Position) bool {
	return m.defenseXSign(gi)*attackerPos.X > attackerThreatX
}

func (m *CombinedPlan) calcGoalWallPositionsForRobots(gi *GameInfo, threatPos Position, count int) []Position {
	if count == 0 {
		return nil
	}

	goalCenter := m.defendedGoalCenter(gi)
	xSign := m.defenseXSign(gi)
	wallX := goalCenter.X - xSign*goalWallDistanceFromGoal
	wallY := threatPos.Y

	dx := goalCenter.X - threatPos.X
	if math.Abs(dx) > 1 {
		t := (wallX - threatPos.X) / dx
		wallY = threatPos.Y + (goalCenter.Y-threatPos.Y)*t
	}
	wallY = clampFloat(wallY, -goalWallYClamp, goalWallYClamp)

	positions := make([]Position, count)
	centerOffset := float64(count-1) / 2.0
	for i := 0; i < count; i++ {
		y := wallY + (float64(i)-centerOffset)*goalWallSpacing
		pos := Position{
			X: wallX,
			Y: clampFloat(y, -goalWallYClamp, goalWallYClamp),
			Z: 0,
		}
		pos.Angle = pos.AngleToPosition(threatPos)
		positions[i] = pos
	}

	return positions
}

func (m *CombinedPlan) run() {
	gi := <-m.incomingGameInfo
	roleManager := newCombinedRoleManager(&m.ActivityHandler, &gi, m.team)
	possessionTracker := &ballPossessionTracker{}
	actorTracker := &offenseBallActorTracker{}
	frameMonitor := helper.NewFrameSkipMonitor(m.team.String() + " combined_plan")
	mode := tacticalModeAttack
	candidateMode := tacticalMode("")
	candidateModeSince := time.Time{}
	//clearTarget := info.Position{X: 2000, Y: -1500, Z: 0, Angle: 0}

	//var goalieID info.ID
	var goalieRole *roles.GoalieRole
	var activeReceiver info.ID
	var activeReceiverStart time.Time
	hasActiveReceiver := false

	for m.Active {
		tickStart := time.Now()
		gi = <-m.incomingGameInfo
		frameMonitor.Observe(gi.VisionFrame())
		possession := possessionTracker.update(&gi, tickStart)
		rawOwner := observedBallOwner(&gi)

		activeRobots := m.activeRobots(&gi)
		//nextGoalieID, hasGoalie := m.chooseGoalie(&gi, activeRobots)
		fieldRobots := activeRobots
		//if hasGoalie {
		//	fieldRobots = withoutRobot(activeRobots, nextGoalieID)
		//	if goalieRole == nil || goalieID != nextGoalieID {
		//		if goalieRole != nil {
		//			m.ActivityHandler.ClearActivity(goalieID)
		//		}
		//		goalieID = nextGoalieID
		//		goalieRole = roles.NewGoalieRole(goalieID, m.ActivityHandler, m.team, clearTarget)
		//		goalieRole.Init()
		//	}
		//} else if goalieRole != nil {
		//	m.ActivityHandler.ClearActivity(goalieID)
		//	goalieRole = nil
		//}

		desiredMode := desiredModeForTrackedOwner(possession.owner, m.team, mode)
		mode = stableMode(mode, desiredMode, &candidateMode, &candidateModeSince, tickStart)

		assignment := roleAssignmentForMode(len(fieldRobots), mode)
		slotAssignments := m.assignTacticalSlots(&gi, fieldRobots, assignment, possession.owner)
		roleManager.applySlotAssignments(slotAssignments, tickStart)
		offenseRobots := roleManager.offenseIDs()
		receiverIDs := roleManager.idsForSlot(tacticalSlotBallReceiver)
		chaserID, hasChaser := roleManager.idForSlot(tacticalSlotBallChaser)
		roleManager.configureOffenseReceivers(receiverIDs)

		ballVel, ballVelOK := gi.State.GetTrackedBall().GetTrackedVelocity()
		ballMoving := ballVelOK && ballVel.Norm2d() > 0.3
		ownerRetained := possession.owner.valid && ownerStillRetained(&gi, possession.owner)

		handledByOffense := false
		passInFlight := hasActiveReceiver &&
			containsRobot(offenseRobots, activeReceiver) &&
			time.Since(activeReceiverStart) < 2*time.Second &&
			(!rawOwner.valid || rawOwner.team != m.team) &&
			(ballMoving || !ownerRetained)
		if !passInFlight && possession.owner.valid && possession.owner.team == m.team {
			ownerID := possession.owner.id
			owner, ok := roleManager.attackers[ownerID]
			if ok {
				handledByOffense = true
				hasActiveReceiver = false

				actorTracker.switchTo(roleManager.attackers, newOffenseBallActor(ownerID, offenseBallActorOwner))
				owner.TriggerEvent("BALL_OWNER")
				decision := owner.CurrentDecision()
				if decision.IsPass && decision.ReceiverID != ownerID {
					receiver, ok := roleManager.attackers[decision.ReceiverID]
					if ok {
						activeReceiver = decision.ReceiverID
						activeReceiverStart = time.Now()
						hasActiveReceiver = true
						receiver.ReceivePass(decision.Target)
					}
				}
			}
		}

		if !handledByOffense {
			if passInFlight {
				actorTracker.switchTo(roleManager.attackers, newOffenseBallActor(activeReceiver, offenseBallActorReceiver))
			} else {
				hasActiveReceiver = false
				interceptorID := chaserID
				if !hasChaser {
					interceptorID = m.getRobotForBall(&gi, offenseRobots)
				}
				if interceptor, ok := roleManager.attackers[interceptorID]; ok {
					actorTracker.switchTo(roleManager.attackers, newOffenseBallActor(interceptorID, offenseBallActorChaser))
					interceptor.TriggerEvent("BALL_APPROACHING")
				} else {
					actorTracker.switchTo(roleManager.attackers, noOffenseBallActor())
				}
			}
		}

		for _, attacker := range roleManager.attackers {
			attacker.Run()
		}

		ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
		threatPos := ballPos
		if attackerPos, found := m.getAttackerPosition(&gi); found && m.attackerIsThreatening(&gi, attackerPos) {
			threatPos = attackerPos
		}

		wallRobots := roleManager.idsForSlot(tacticalSlotDefenderWall)
		wallPositions := m.calcGoalWallPositionsForRobots(&gi, threatPos, len(wallRobots))
		for i, id := range wallRobots {
			if defender, ok := roleManager.defenders[id]; ok {
				defender.SetWallPosition(wallPositions[i])
				defender.TriggerEvent("ATTACKER_NEAR")
			}
		}
		for id, defender := range roleManager.defenders {
			if containsRobot(wallRobots, id) {
				continue
			}
			defender.TriggerEvent("ATTACKER_FAR")
		}
		for _, d := range roleManager.defenders {
			d.Run()
		}

		if goalieRole != nil {
			goalieRole.SetGameInfo(gi)
			if goalieRole.ShouldClearBall(roles.GoalieBallControlRadius, attackerThreatX) {
				goalieRole.TriggerEvent("BALL_OWNER")
			} else {
				goalieRole.TriggerEvent("BALL_LOST")
			}
			goalieRole.Run()
		}
	}
}

func (m *CombinedPlan) Kill() {
	m.Active = false
}
