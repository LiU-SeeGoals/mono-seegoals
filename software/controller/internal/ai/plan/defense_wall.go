package plan

import (
	"math"

	"github.com/LiU-SeeGoals/controller/internal/debugstate"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
	"github.com/LiU-SeeGoals/controller/internal/rules"
)

const (
	// Opponents closer than this (mm, along the field axis) to our goal line
	// count as a threat to our goal (2800 mm from center on a 9 m field).
	attackerThreatDistanceFromGoal = 1700.0

	goalWallYClamp  = 1200.0
	goalWallSpacing = 240.0
)

// attackerThreatDepth is the X depth (unsigned, from the center line) past
// which an opponent threatens our goal.
func attackerThreatDepth(gi *info.GameInfo) float64 {
	return gi.FieldHalfLength() - attackerThreatDistanceFromGoal
}

func (m *CombinedPlan) attackerIsThreatening(gi *info.GameInfo, attackerPos info.Position) bool {
	return gi.OwnHalfXSign(m.team)*attackerPos.X > attackerThreatDepth(gi)
}

// getAttackerPosition returns the opponent closest to the ball.
func (m *CombinedPlan) getAttackerPosition(gi *info.GameInfo) (info.Position, bool) {
	opponents := gi.State.GetOtherTeam(m.team)
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()

	minDist := math.Inf(1)
	var closestPos info.Position
	found := false

	var i info.ID
	for i = 0; i < info.TEAM_SIZE; i++ {
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

// calcGoalWallPositionsForRobots lines count robots up on the defense line
// (just outside our defense area), centered on the threat-goal line.
func (m *CombinedPlan) calcGoalWallPositionsForRobots(gi *info.GameInfo, threatPos info.Position, count int) []info.Position {
	if count == 0 {
		return nil
	}

	goalCenter := gi.OwnGoalCenter(m.team)
	xSign := gi.OwnHalfXSign(m.team)
	wallX := xSign * rules.DefenseLineDepth(gi)
	wallY := threatPos.Y

	dx := goalCenter.X - threatPos.X
	if math.Abs(dx) > 1 {
		t := (wallX - threatPos.X) / dx
		wallY = threatPos.Y + (goalCenter.Y-threatPos.Y)*t
	}
	wallY = clampFloat(wallY, -goalWallYClamp, goalWallYClamp)

	positions := make([]info.Position, count)
	centerOffset := float64(count-1) / 2.0
	for i := 0; i < count; i++ {
		y := wallY + (float64(i)-centerOffset)*goalWallSpacing
		pos := info.Position{
			X: wallX,
			Y: clampFloat(y, -goalWallYClamp, goalWallYClamp),
			Z: 0,
		}
		pos.Angle = pos.AngleToPosition(threatPos)
		positions[i] = pos
	}

	return positions
}

// goalieClearTarget is where the goalie clears to when it finds no safe pass:
// into the opponent half, mirrored with the side we defend.
func (m *CombinedPlan) goalieClearTarget(gi *info.GameInfo) info.Position {
	return info.Position{X: -gi.OwnHalfXSign(m.team) * 2000, Y: -1500}
}

func (m *CombinedPlan) updateDefenseAndGoaliePositioning(stopped bool) {
	gi := &m.gi
	rm := m.roleManager

	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()
	threatPos := ballPos
	if stopped && gi.Status.GetGameEvent().GetDesignatedPosition() != nil {
		pos := gi.Status.GetGameEvent().GetDesignatedPosition()
		threatPos = info.Position{X: pos.AtVec(0), Y: pos.AtVec(1)}
	} else if attackerPos, found := m.getAttackerPosition(gi); !stopped && found && m.attackerIsThreatening(gi, attackerPos) {
		threatPos = attackerPos
	}

	wallRobots := rm.idsForSlot(tacticalSlotDefenderWall)
	wallPositions := m.calcGoalWallPositionsForRobots(gi, threatPos, len(wallRobots))
	for i, id := range wallRobots {
		if defender, ok := rm.defenders[id]; ok {
			defender.SetWallPosition(wallPositions[i])
			defender.TriggerEvent(roles.EventAttackerNear)
		}
	}
	for id, defender := range rm.defenders {
		if containsRobot(wallRobots, id) {
			continue
		}
		defender.TriggerEvent(roles.EventAttackerFar)
	}
	for _, defender := range rm.defenders {
		defender.Run()
	}

	goalieRole := m.goalieRole
	if goalieRole == nil {
		return
	}

	debugstate.SetRobotRole(m.team, m.goalieID, tacticalSlotGoalie.label())
	goalieRole.SetGameInfo(*gi)
	goalieRole.SetClearFallback(m.goalieClearTarget(gi))
	if stopped {
		goalieRole.TriggerEvent(roles.EventBallLost)
	} else if goalieRole.ShouldClearBall(roles.GoalieBallControlRadius, attackerThreatDepth(gi)) {
		goalieRole.TriggerEvent(roles.EventBallOwner)
	} else if goalieRole.ShouldCollectDeadBall() {
		goalieRole.TriggerEvent(roles.EventDeadBallTrapped)
	} else if !goalieRole.IsDeadBallRescueActive() {
		goalieRole.TriggerEvent(roles.EventBallLost)
	}
	goalieRole.Run()
}
