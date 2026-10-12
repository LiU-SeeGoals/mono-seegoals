package ai

import (
	"math"
	"sort"

	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

// Distance still matters when a defender first chooses a mark, but threat to
// our goal remains the stronger part of the decision. Once assigned, the score
// is no longer consulted until the opponent leaves the dangerous region.
const coverDefenderDistanceWeight = 0.35

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
		if _, _, dangerous := roles.LockedCoverThreat(gi, team, opponentID, ballPos); !dangerous || usedOpponents[opponentID] {
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

	// Allocate remaining defenders and opponents as pairs. This avoids the first
	// defender in ID order consuming the natural target of another defender.
	for {
		bestDefender := info.ID(0)
		bestOpponent := info.ID(0)
		bestScore := math.Inf(1)
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
				_, opponentPos, dangerous := roles.CoverThreat(gi, team, opponentID, ballPos)
				if !dangerous {
					continue
				}

				goalDistance := opponentPos.Dist2d(ownGoalCenter(gi, team))
				defenderDistance := defenderPos.Dist2d(opponentPos)
				score := goalDistance + coverDefenderDistanceWeight*defenderDistance
				if !found || score < bestScore ||
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

func ownGoalCenter(gi *info.GameInfo, team info.Team) info.Position {
	halfFieldLength := 4500.0
	if gi != nil && gi.HasField() {
		fieldSize := gi.FieldSize()
		if fieldSize.X > 0 {
			halfFieldLength = fieldSize.X / 2
		}
	}
	return info.Position{X: gi.OwnHalfXSign(team) * halfFieldLength}
}
