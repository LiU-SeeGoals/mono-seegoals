package plan

import (
	"math"

	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/referee"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

const (
	activeRobotTimeoutMillis = int64(1000)
	lookaheadSec             = 0.5
)

type roleAssignment struct {
	slots []tacticalSlotKind
}

type tacticalSlotKind string

const (
	tacticalSlotBallChaser     tacticalSlotKind = "ball_chaser"
	tacticalSlotSupportShooter tacticalSlotKind = "support_shooter"
	tacticalSlotDefenderWall   tacticalSlotKind = "defender_wall"
	tacticalSlotDefenderHigh   tacticalSlotKind = "defender_high"
	tacticalSlotDefenderLow    tacticalSlotKind = "defender_low"
	tacticalSlotGoalie         tacticalSlotKind = "goalie"
)

func (slot tacticalSlotKind) roleKind() roleKind {
	switch slot {
	case tacticalSlotBallChaser, tacticalSlotSupportShooter:
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
	case tacticalSlotSupportShooter:
		return "Shooter"
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

// defenseSlotKind maps a defender slot onto the DefenseRole kind that drives it.
func (slot tacticalSlotKind) defenseSlotKind() (roles.DefenseRoleKind, bool) {
	switch slot {
	case tacticalSlotDefenderWall:
		return roles.DefenseRoleWall, true
	case tacticalSlotDefenderHigh:
		return roles.DefenseRoleHigh, true
	case tacticalSlotDefenderLow:
		return roles.DefenseRoleLow, true
	default:
		return "", false
	}
}

// roleAssignmentForMode lists the slots for total field robots, in priority
// order. Up to five robots get a hand-tuned formation; each robot beyond that
// adds another goal-wall defender (defend) or support shooter (attack), so
// larger divisions do not leave robots idle.
func roleAssignmentForMode(total int, mode tacticalMode) roleAssignment {
	var base []tacticalSlotKind
	extra := tacticalSlotSupportShooter
	if mode == tacticalModeDefend {
		extra = tacticalSlotDefenderWall
		base = []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotSupportShooter, tacticalSlotDefenderWall, tacticalSlotDefenderLow, tacticalSlotDefenderHigh}
		switch total {
		case 3:
			base = []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotDefenderWall, tacticalSlotDefenderLow}
		case 4:
			base = []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotDefenderWall, tacticalSlotDefenderLow, tacticalSlotDefenderHigh}
		}
	} else {
		base = []tacticalSlotKind{tacticalSlotBallChaser, tacticalSlotSupportShooter, tacticalSlotSupportShooter, tacticalSlotDefenderLow, tacticalSlotDefenderHigh}
	}

	if total <= 0 {
		return roleAssignment{}
	}
	if total <= len(base) {
		return roleAssignment{slots: append([]tacticalSlotKind{}, base[:total]...)}
	}
	slots := append([]tacticalSlotKind{}, base...)
	for len(slots) < total {
		slots = append(slots, extra)
	}
	return roleAssignment{slots: slots}
}

func (m *CombinedPlan) assignTacticalSlots(
	gi *info.GameInfo, robots []info.ID,
	assignment roleAssignment,
	owner trackedBallOwner,
	restrictedRobot info.ID,
	hasRestrictedRobot bool,
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
			ballChaserCandidates := available
			if hasRestrictedRobot {
				ballChaserCandidates = withoutRobot(ballChaserCandidates, restrictedRobot)
			}
			if len(ballChaserCandidates) == 0 {
				continue
			}
			id = m.chooseBallChaser(gi, ballChaserCandidates, owner)
		case tacticalSlotSupportShooter:
			id = m.chooseSupportShooter(gi, available)
		default:
			id = m.getRobotClosestToPosition(gi, available, m.targetForTacticalSlot(gi, slot))
		}

		assignments[id] = slot
		available = withoutRobot(available, id)
	}

	return assignments
}

// targetForTacticalSlot is where a robot in slot will drive, using the same
// calculation as the role that drives it.
func (m *CombinedPlan) targetForTacticalSlot(gi *info.GameInfo, slot tacticalSlotKind) info.Position {
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()

	if slot == tacticalSlotDefenderWall {
		if wall := m.calcGoalWallPositionsForRobots(gi, ballPos, 1); len(wall) > 0 {
			return wall[0]
		}
		return ballPos
	}
	if kind, ok := slot.defenseSlotKind(); ok {
		return roles.DefenderTarget(gi, m.team, roles.DefenseSlot{Kind: kind}, ballPos)
	}
	return ballPos
}

func (m *CombinedPlan) chooseBallChaser(gi *info.GameInfo, candidates []info.ID, owner trackedBallOwner) info.ID {
	if owner.valid && owner.team == m.team && containsRobot(candidates, owner.id) {
		return owner.id
	}

	possessor := gi.State.GetBall().GetPossessor()
	if possessor != nil && possessor.GetTeam() == m.team && containsRobot(candidates, possessor.GetID()) {
		return possessor.GetID()
	}
	return m.getRobotForBall(gi, candidates)
}

func (m *CombinedPlan) chooseSupportShooter(gi *info.GameInfo, candidates []info.ID) info.ID {
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
	targetPos := activity.PredictedBallPos(gi, lookaheadSec)
	return m.getRobotClosestToPosition(gi, activeRobots, targetPos)
}

// activeRobots lists our robots seen within activeRobotTimeoutMillis, or every
// robot ever seen when none is recent (e.g. a vision dropout).
func (m *CombinedPlan) activeRobots(gi *info.GameInfo) []info.ID {
	team := gi.State.GetTeam(m.team)
	nowMillis := gi.Now().UnixMilli()
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

// splitGoalieFromFieldRobots elects the goalie with the same rule the referee
// restarts use, so the goalie does not change hands at a restart.
func (m *CombinedPlan) splitGoalieFromFieldRobots(
	gi *info.GameInfo, activeRobots []info.ID,
) (info.ID, bool, []info.ID) {
	goalieID, hasGoalie := referee.SelectGoalieID(gi, m.team, activeRobots)
	if !hasGoalie {
		return 0, false, activeRobots
	}
	return goalieID, true, withoutRobot(activeRobots, goalieID)
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

func clampFloat(value float64, minValue float64, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}
