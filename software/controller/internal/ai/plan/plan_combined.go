package ai

import (
	"math"
	"sort"
	"sync"
	"time"

	coreai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
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

	attackModeAttackerRatio = 0.50
	defendModeAttackerRatio = 0.20

	roleSwitchMinDuration    = 750 * time.Millisecond
	tacticalModeSwitchDelay  = 500 * time.Millisecond
	activeRobotTimeoutMillis = int64(1000)

	preferredGoalieID = info.ID(6)
)

type roleKind string

const (
	roleOffense roleKind = "offense"
	roleDefense roleKind = "defense"
)

type combinedRoleManager struct {
	activityHandler *coreai.ActivityHandler
	gi              *GameInfo
	team            Team
	kickers         map[info.ID]*roles.OffenseRole
	defenders       map[info.ID]*roles.DefenseRole
	roleByRobot     map[info.ID]roleKind
	lastChanged     map[info.ID]time.Time
}

func newCombinedRoleManager(activityHandler *coreai.ActivityHandler, gi *GameInfo, team Team) *combinedRoleManager {
	return &combinedRoleManager{
		activityHandler: activityHandler,
		gi:              gi,
		team:            team,
		kickers:         make(map[info.ID]*roles.OffenseRole),
		defenders:       make(map[info.ID]*roles.DefenseRole),
		roleByRobot:     make(map[info.ID]roleKind),
		lastChanged:     make(map[info.ID]time.Time),
	}
}

func (rm *combinedRoleManager) applyAssignments(offenseIDs []info.ID, defenseIDs []info.ID, now time.Time) {
	desired := make(map[info.ID]roleKind)
	for _, id := range offenseIDs {
		desired[id] = roleOffense
	}
	for _, id := range defenseIDs {
		desired[id] = roleDefense
	}

	for id := range rm.roleByRobot {
		if _, ok := desired[id]; !ok {
			rm.remove(id)
		}
	}

	for id, role := range desired {
		current, exists := rm.roleByRobot[id]
		if exists && current == role {
			continue
		}
		if exists && now.Sub(rm.lastChanged[id]) < roleSwitchMinDuration {
			continue
		}

		switch role {
		case roleOffense:
			rm.assignOffense(id, now)
		case roleDefense:
			rm.assignDefense(id, now)
		}
	}
}

func (rm *combinedRoleManager) assignOffense(id info.ID, now time.Time) {
	rm.remove(id)
	kicker := roles.NewOffenseRole(id, *rm.activityHandler, rm.gi, rm.team)
	kicker.Init()
	rm.kickers[id] = kicker
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
	delete(rm.kickers, id)
	delete(rm.defenders, id)
	delete(rm.roleByRobot, id)
	rm.activityHandler.ClearActivity(id)
}

func (rm *combinedRoleManager) offenseIDs() []info.ID {
	ids := make([]info.ID, 0, len(rm.kickers))
	for id := range rm.kickers {
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
	return 0, false

	// if containsRobot(activeRobots, preferredGoalieID) {
	// 	return preferredGoalieID, true
	// }
	// if len(activeRobots) == 0 {
	// 	return 0, false
	// }
	//
	// return m.getRobotClosestToPosition(gi, activeRobots, m.defendedGoalCenter(gi)), true
}

func attackerCount(total int, ratio float64) int {
	if total == 0 {
		return 0
	}

	count := int(math.Ceil(float64(total) * ratio))
	if count < 1 {
		return 1
	}
	if count > total {
		return total
	}
	return count
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

func (m *CombinedPlan) selectOffenseRobots(gi *GameInfo, candidates []info.ID, count int) []info.ID {
	if count >= len(candidates) {
		return append([]info.ID{}, candidates...)
	}

	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	ballVel, ok := gi.State.GetTrackedBall().GetTrackedVelocity()
	targetPos := ballPos
	if ok && ballVel.Norm2d() > 0.3 {
		lookaheadSeconds := 0.5
		targetPos.X = ballPos.X + ballVel.X*1000*lookaheadSeconds
		targetPos.Y = ballPos.Y + ballVel.Y*1000*lookaheadSeconds
	}

	sorted := append([]info.ID{}, candidates...)
	sort.SliceStable(sorted, func(i, j int) bool {
		posI, errI := gi.State.GetRobotPosition(m.team, sorted[i])
		posJ, errJ := gi.State.GetRobotPosition(m.team, sorted[j])
		if errI != nil {
			return false
		}
		if errJ != nil {
			return true
		}
		return posI.Dist2d(targetPos) < posJ.Dist2d(targetPos)
	})

	return sorted[:count]
}

func splitRoles(allRobots []info.ID, offenseRobots []info.ID) []info.ID {
	defenseRobots := []info.ID{}
	for _, id := range allRobots {
		if !containsRobot(offenseRobots, id) {
			defenseRobots = append(defenseRobots, id)
		}
	}
	return defenseRobots
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

func (m *CombinedPlan) calcWallPositions(attackerPos Position, goalCenter Position) (Position, Position, Position) {
	shotVec := goalCenter.Sub(&attackerPos)
	shotVecNorm := shotVec.Normalize2d()

	mid := attackerPos.Add(&goalCenter)
	mid.Div2d(2.0)

	perpVec := Position{X: -shotVecNorm.Y, Y: shotVecNorm.X, Z: 0, Angle: 0}

	offset := perpVec.Scale(wallSpacing)
	bot1Pos := mid.Sub(&offset) // one side
	bot2Pos := mid              // center
	bot3Pos := mid.Add(&offset) // other side

	bot1Pos.Angle = bot1Pos.AngleToPosition(attackerPos)
	bot2Pos.Angle = bot2Pos.AngleToPosition(attackerPos)
	bot3Pos.Angle = bot3Pos.AngleToPosition(attackerPos)

	return bot1Pos, bot2Pos, bot3Pos
}

func (m *CombinedPlan) calcWallPositionsForRobots(attackerPos Position, goalCenter Position, count int) []Position {
	if count == 0 {
		return nil
	}

	shotVec := goalCenter.Sub(&attackerPos)
	shotVecNorm := shotVec.Normalize2d()

	mid := attackerPos.Add(&goalCenter)
	mid.Div2d(2.0)

	perpVec := Position{X: -shotVecNorm.Y, Y: shotVecNorm.X, Z: 0, Angle: 0}
	positions := make([]Position, count)
	centerOffset := float64(count-1) / 2.0

	for i := 0; i < count; i++ {
		offset := perpVec.Scale((float64(i) - centerOffset) * wallSpacing)
		pos := mid.Add(&offset)
		pos.Angle = pos.AngleToPosition(attackerPos)
		positions[i] = pos
	}

	return positions
}

func (m *CombinedPlan) run() {
	gi := <-m.incomingGameInfo
	roleManager := newCombinedRoleManager(&m.ActivityHandler, &gi, m.team)
	mode := tacticalModeAttack
	candidateMode := tacticalMode("")
	candidateModeSince := time.Time{}
	clearTarget := info.Position{X: 2000, Y: -1500, Z: 0, Angle: 0}

	var goalieID info.ID
	var goalieRole *roles.GoalieRole
	var activeReceiver info.ID
	var activeReceiverStart time.Time
	hasActiveReceiver := false

	for m.Active {
		tickStart := time.Now()
		gi = <-m.incomingGameInfo

		activeRobots := m.activeRobots(&gi)
		nextGoalieID, hasGoalie := m.chooseGoalie(&gi, activeRobots)
		fieldRobots := activeRobots
		if hasGoalie {
			fieldRobots = withoutRobot(activeRobots, nextGoalieID)
			if goalieRole == nil || goalieID != nextGoalieID {
				if goalieRole != nil {
					m.ActivityHandler.ClearActivity(goalieID)
				}
				goalieID = nextGoalieID
				goalieRole = roles.NewGoalieRole(goalieID, m.ActivityHandler, m.team, clearTarget)
				goalieRole.Init()
			}
		} else if goalieRole != nil {
			m.ActivityHandler.ClearActivity(goalieID)
			goalieRole = nil
		}

		desiredMode := m.desiredMode(&gi, mode)
		mode = stableMode(mode, desiredMode, &candidateMode, &candidateModeSince, tickStart)

		attackerRatio := attackModeAttackerRatio
		if mode == tacticalModeDefend {
			attackerRatio = defendModeAttackerRatio
		}

		offenseRobots := m.selectOffenseRobots(&gi, fieldRobots, attackerCount(len(fieldRobots), attackerRatio))
		defenseRobots := splitRoles(fieldRobots, offenseRobots)
		roleManager.applyAssignments(offenseRobots, defenseRobots, tickStart)
		offenseRobots = roleManager.offenseIDs()
		defenseRobots = roleManager.defenseIDs()

		possessor := gi.State.GetBall().GetPossessor()

		handledByOffense := false
		if possessor != nil && possessor.GetTeam() == m.team {
			ownerID := possessor.GetID()
			owner, ok := roleManager.kickers[ownerID]
			if ok {
				handledByOffense = true
				hasActiveReceiver = false
				for _, id := range offenseRobots {
					if id != ownerID {
						if kicker, ok := roleManager.kickers[id]; ok {
							kicker.TriggerEvent("BALL_LOST")
						}
					}
				}

				owner.TriggerEvent("BALL_OWNER")
				decision := owner.CurrentDecision()
				if decision.IsPass && decision.ReceiverID != ownerID {
					receiver, ok := roleManager.kickers[decision.ReceiverID]
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
			if hasActiveReceiver && containsRobot(offenseRobots, activeReceiver) && time.Since(activeReceiverStart) < 2*time.Second {
				for _, id := range offenseRobots {
					if id != activeReceiver {
						if kicker, ok := roleManager.kickers[id]; ok {
							kicker.TriggerEvent("BALL_LOST")
						}
					}
				}
			} else {
				hasActiveReceiver = false
				interceptorID := m.getRobotForBall(&gi, offenseRobots)
				for _, id := range offenseRobots {
					if id != interceptorID {
						if kicker, ok := roleManager.kickers[id]; ok {
							kicker.TriggerEvent("BALL_LOST")
						}
					}
				}
				if interceptor, ok := roleManager.kickers[interceptorID]; ok {
					interceptor.TriggerEvent("BALL_APPROACHING")
				}
			}
		}

		for _, kicker := range roleManager.kickers {
			kicker.Run()
		}

		attackerPos, found := m.getAttackerPosition(&gi)
		if found && m.attackerIsThreatening(&gi, attackerPos) {
			wallPositions := m.calcWallPositionsForRobots(attackerPos, m.defendedGoalCenter(&gi), len(defenseRobots))
			for i, id := range defenseRobots {
				if defender, ok := roleManager.defenders[id]; ok {
					defender.SetWallPosition(wallPositions[i])
				}
			}
			for _, d := range roleManager.defenders {
				d.TriggerEvent("ATTACKER_NEAR")
			}
		} else {
			for _, d := range roleManager.defenders {
				d.TriggerEvent("ATTACKER_FAR")
			}
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

		helper.PaceLoop(tickStart, helper.PlannerLoopPeriod, "combined_plan")
	}
}

func (m *CombinedPlan) Kill() {
	m.Active = false
}
