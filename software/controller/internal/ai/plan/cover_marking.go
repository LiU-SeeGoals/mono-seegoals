package ai

import (
	"math"
	"sort"

	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

const (
	// The weights deliberately sum to one. Keeping every component normalized
	// makes it possible to tune one defensive preference without mixing metres,
	// millimetres and radians in the same score.
	receiverGoalProximityWeight = 0.35
	receiverCentralityWeight    = 0.15
	receiverPassLineWeight      = 0.30
	receiverGoalAngleWeight     = 0.20

	// Defender distance is only a small tie-breaker when allocating several
	// defenders. The opponent's threat score should decide who is marked.
	receiverDefenderDistanceWeight = 0.10
	receiverSwitchMargin           = 0.10

	// A pass lane is treated as blocked near a robot body and fully open once
	// the nearest defender is this far from the ball-to-receiver segment.
	receiverClosedLaneClearance = 120.0
	receiverOpenLaneClearance   = 700.0
)

type receiverThreatScore struct {
	Total            float64
	GoalProximity    float64
	Centrality       float64
	PassLineOpenness float64
	OpenGoalAngle    float64
}

type coverMarkingTracker struct {
	assignments map[info.ID]info.ID
}

func (t *coverMarkingTracker) update(
	gi *info.GameInfo,
	team info.Team,
	defenderIDs []info.ID,
) map[info.ID]info.ID {
	if t.assignments == nil {
		t.assignments = make(map[info.ID]info.ID)
	}
	if gi == nil || gi.State == nil {
		clear(t.assignments)
		return t.assignments
	}
	if _, ok := gi.FieldGeometry(); !ok {
		clear(t.assignments)
		return t.assignments
	}
	ballPos, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		clear(t.assignments)
		return t.assignments
	}

	ids := append([]info.ID(nil), defenderIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	activeDefenders := make(map[info.ID]bool, len(ids))
	for _, id := range ids {
		activeDefenders[id] = true
	}

	// Keep valid locks. Sorting makes recovery deterministic if stale state ever
	// contains two defenders assigned to the same opponent.
	usedOpponents := make(map[info.ID]bool)
	for _, defenderID := range ids {
		opponentID, assigned := t.assignments[defenderID]
		if !assigned {
			continue
		}
		if _, err := gi.State.GetRobotPosition(team, defenderID); err != nil {
			delete(t.assignments, defenderID)
			continue
		}
		if _, _, dangerous := roles.LockedPassReceiverThreat(gi, team, opponentID, ballPos); !dangerous || usedOpponents[opponentID] {
			delete(t.assignments, defenderID)
			continue
		}
		usedOpponents[opponentID] = true
	}
	for defenderID := range t.assignments {
		if !activeDefenders[defenderID] {
			delete(t.assignments, defenderID)
		}
	}

	// Keep the stable mark for small score changes, but release it when another
	// unmarked receiver is substantially more dangerous. This preserves the
	// anti-jitter lock without making the initial choice permanent.
	for _, defenderID := range ids {
		currentOpponentID, assigned := t.assignments[defenderID]
		if !assigned {
			continue
		}
		defenderPos, err := gi.State.GetRobotPosition(team, defenderID)
		if err != nil {
			continue
		}
		_, currentOpponentPos, valid := roles.LockedPassReceiverThreat(gi, team, currentOpponentID, ballPos)
		if !valid {
			continue
		}
		currentScore := receiverAssignmentScore(gi, team, defenderID, defenderPos, ballPos, currentOpponentPos)
		bestAlternativeScore := math.Inf(-1)
		for opponentIndex := range gi.State.GetOtherTeam(team) {
			opponentID := info.ID(opponentIndex)
			if usedOpponents[opponentID] {
				continue
			}
			_, opponentPos, dangerous := roles.PassReceiverThreat(gi, team, opponentID, ballPos)
			if !dangerous {
				continue
			}
			score := receiverAssignmentScore(gi, team, defenderID, defenderPos, ballPos, opponentPos)
			if score > bestAlternativeScore {
				bestAlternativeScore = score
			}
		}
		if bestAlternativeScore > currentScore+receiverSwitchMargin {
			delete(t.assignments, defenderID)
			delete(usedOpponents, currentOpponentID)
		}
	}

	// Allocate remaining defenders and opponents as pairs. This avoids the first
	// defender in ID order consuming the natural target of another defender.
	for {
		bestDefender := info.ID(0)
		bestOpponent := info.ID(0)
		bestScore := math.Inf(-1)
		found := false

		for _, defenderID := range ids {
			if _, assigned := t.assignments[defenderID]; assigned {
				continue
			}
			defenderPos, err := gi.State.GetRobotPosition(team, defenderID)
			if err != nil {
				continue
			}

			for opponentIndex := range gi.State.GetOtherTeam(team) {
				opponentID := info.ID(opponentIndex)
				if usedOpponents[opponentID] {
					continue
				}
				_, opponentPos, dangerous := roles.PassReceiverThreat(gi, team, opponentID, ballPos)
				if !dangerous {
					continue
				}

				score := receiverAssignmentScore(gi, team, defenderID, defenderPos, ballPos, opponentPos)
				if !found || score > bestScore ||
					(score == bestScore && (defenderID < bestDefender ||
						(defenderID == bestDefender && opponentID < bestOpponent))) {
					bestDefender = defenderID
					bestOpponent = opponentID
					bestScore = score
					found = true
				}
			}
		}

		if !found {
			break
		}
		t.assignments[bestDefender] = bestOpponent
		usedOpponents[bestOpponent] = true
	}

	return t.assignments
}

func receiverAssignmentScore(
	gi *info.GameInfo,
	team info.Team,
	defenderID info.ID,
	defenderPos info.Position,
	ballPos info.Position,
	opponentPos info.Position,
) float64 {
	threat := scorePassReceiver(gi, team, defenderID, ballPos, opponentPos)
	defenderDistance := defenderPos.Dist2d(opponentPos)
	geometry, ok := gi.FieldGeometry()
	if !ok {
		return math.Inf(-1)
	}
	fieldDiagonal := math.Hypot(geometry.Length, geometry.Width)
	if fieldDiagonal <= 0 {
		return math.Inf(-1)
	}
	defenderPenalty := receiverDefenderDistanceWeight * clampUnit(defenderDistance/fieldDiagonal)
	return threat.Total - defenderPenalty
}

// scorePassReceiver gives a candidate pass receiver a normalized threat score.
// Higher means more dangerous. ignoredDefender is excluded from the pass-lane
// check because that robot is the one being considered for the assignment; its
// current position must not make an otherwise open receiver look covered.
func scorePassReceiver(
	gi *info.GameInfo,
	team info.Team,
	ignoredDefender info.ID,
	ballPos info.Position,
	receiverPos info.Position,
) receiverThreatScore {
	if gi == nil {
		return receiverThreatScore{}
	}
	geometry, ok := gi.FieldGeometry()
	if !ok {
		return receiverThreatScore{}
	}
	goalPos, ok := gi.OwnGoalCenter(team)
	if !ok {
		return receiverThreatScore{}
	}
	halfFieldLength := geometry.Length / 2
	halfFieldWidth := geometry.Width / 2
	halfGoalWidth := geometry.GoalWidth / 2
	if halfFieldLength <= 0 || halfFieldWidth <= 0 || halfGoalWidth <= 0 {
		return receiverThreatScore{}
	}

	goalProximity := 1 - clampUnit(receiverPos.Dist2d(goalPos)/halfFieldLength)
	centrality := 1 - clampUnit(math.Abs(receiverPos.Y)/halfFieldWidth)
	passLineOpenness := passLaneOpenness(gi, team, ignoredDefender, ballPos, receiverPos)
	openGoalAngle := clampUnit(goalOpeningAngle(receiverPos, goalPos.X, halfGoalWidth) / (math.Pi / 2))

	return receiverThreatScore{
		Total: receiverGoalProximityWeight*goalProximity +
			receiverCentralityWeight*centrality +
			receiverPassLineWeight*passLineOpenness +
			receiverGoalAngleWeight*openGoalAngle,
		GoalProximity:    goalProximity,
		Centrality:       centrality,
		PassLineOpenness: passLineOpenness,
		OpenGoalAngle:    openGoalAngle,
	}
}

func passLaneOpenness(
	gi *info.GameInfo,
	team info.Team,
	ignoredDefender info.ID,
	ballPos info.Position,
	receiverPos info.Position,
) float64 {
	if gi == nil || gi.State == nil {
		return 1
	}

	minimumClearance := math.Inf(1)
	for robotIndex, robot := range gi.State.GetTeam(team) {
		if info.ID(robotIndex) == ignoredDefender || robot == nil || !robot.IsActive() {
			continue
		}
		robotPos, err := robot.GetPosition()
		if err != nil {
			continue
		}
		clearance := pointToSegmentDistance(robotPos, ballPos, receiverPos)
		if clearance < minimumClearance {
			minimumClearance = clearance
		}
	}
	if math.IsInf(minimumClearance, 1) {
		return 1
	}

	return clampUnit(
		(minimumClearance - receiverClosedLaneClearance) /
			(receiverOpenLaneClearance - receiverClosedLaneClearance),
	)
}

func pointToSegmentDistance(point, start, end info.Position) float64 {
	dx := end.X - start.X
	dy := end.Y - start.Y
	segmentLengthSquared := dx*dx + dy*dy
	if segmentLengthSquared < 1 {
		return point.Dist2d(start)
	}

	t := ((point.X-start.X)*dx + (point.Y-start.Y)*dy) / segmentLengthSquared
	t = clampUnit(t)
	closest := info.Position{X: start.X + t*dx, Y: start.Y + t*dy}
	return point.Dist2d(closest)
}

func goalOpeningAngle(receiver info.Position, goalX, halfGoalWidth float64) float64 {
	upperX := goalX - receiver.X
	upperY := halfGoalWidth - receiver.Y
	lowerX := goalX - receiver.X
	lowerY := -halfGoalWidth - receiver.Y
	cross := upperX*lowerY - upperY*lowerX
	dot := upperX*lowerX + upperY*lowerY
	return math.Abs(math.Atan2(cross, dot))
}

func clampUnit(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
