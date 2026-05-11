package ai

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

const (
	wallSpacing           = 200.0
	minDefenderSeparation = 450.0
	roleStickBonus        = 180.0
	passLaneBlockDepth    = 0.45
)

type TwoBotDefence struct {
	plannerCore
	activeBots []info.ID
}

func NewTwoBotDefence(team info.Team, activeBots []info.ID) *TwoBotDefence {
	return &TwoBotDefence{
		plannerCore: plannerCore{
			team: team,
		},
		activeBots: activeBots,
	}
}

func (m *TwoBotDefence) Kill() {
	fmt.Println("Killing TwoBotDefence planner")
}

func (m *TwoBotDefence) Init(
	incoming <-chan GameInfo,
	activities *[TEAM_SIZE]act.Activity,
	lock *sync.Mutex,
	team Team,
) {
	m.incomingGameInfo = incoming
	m.ActivityHandler.Activities = activities
	m.ActivityHandler.Activity_lock = lock
	m.team = team

	go m.run()
}

func (m *TwoBotDefence) getAttackerPosition(gi *GameInfo) (Position, bool) {
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

func (m *TwoBotDefence) calcWallPositions(attackerPos Position, ownGoalCenter Position, n int) []Position {
	shotVec := ownGoalCenter.Sub(&attackerPos)
	shotVecNorm := shotVec.Normalize2d()

	mid := attackerPos.Add(&ownGoalCenter)
	mid.Div2d(2.0)

	perpVec := Position{X: -shotVecNorm.Y, Y: shotVecNorm.X, Z: 0, Angle: 0}

	positions := make([]Position, n)
	for i := 0; i < n; i++ {
		offsetMult := float64(i) - float64(n-1)/2.0
		scaledOffset := perpVec.Scale(offsetMult * wallSpacing)
		pos := mid.Add(&scaledOffset)
		pos.Angle = pos.AngleToPosition(attackerPos)
		positions[i] = pos
	}
	return positions
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func normalizeSafe(p Position) Position {
	if p.Norm2d() < 1e-6 {
		return Position{}
	}
	return p.Normalize2d()
}

func (m *TwoBotDefence) ownGoalCenterFromEnemyGoal(enemyGoal Position) Position {
	return Position{X: -enemyGoal.X, Y: 0, Z: 0, Angle: 0}
}

func (m *TwoBotDefence) calculateCentralCoverPos(ballPos, ownGoalCenter Position) Position {
	goalToBall := ballPos.Sub(&ownGoalCenter)
	dir := normalizeSafe(goalToBall)
	if dir.Norm2d() < 1e-6 {
		dir = Position{X: 1, Y: 0}
	}
	scaled := dir.Scale(2800.0)
	target := ownGoalCenter.Add(&scaled)
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (m *TwoBotDefence) calculatePressPos(ballPos Position) Position {
	return ballPos
}

func (m *TwoBotDefence) calculateWidePos(ballPos Position, ownGoalCenter Position, side float64) Position {
	enemyGoalX := -ownGoalCenter.X
	sign := math.Copysign(1.0, ownGoalCenter.X)

	targetY := clampF(ballPos.Y+side*900.0, -2500.0, 2500.0)
	targetX := sign * 2000.0

	if math.Abs(ballPos.X-enemyGoalX) < math.Abs(targetX-enemyGoalX) {
		targetX = sign * 2400.0
	}

	target := Position{X: targetX, Y: targetY, Z: 0, Angle: 0}
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (m *TwoBotDefence) calculateSupportPos(ballPos, ownGoalCenter Position, gi *GameInfo) Position {
	intercept, found := m.getMostDangerousPassBlockPoint(gi, ballPos)
	if found {
		intercept.Angle = intercept.AngleToPosition(ballPos)
		return intercept
	}

	sign := math.Copysign(1.0, ownGoalCenter.X)
	targetY := clampF(-0.4*ballPos.Y, -2200.0, 2200.0)
	target := Position{X: sign * 1200.0, Y: targetY, Z: 0, Angle: 0}
	target.Angle = target.AngleToPosition(ballPos)
	return target
}

func (m *TwoBotDefence) getMostDangerousPassBlockPoint(gi *GameInfo, ballPos Position) (Position, bool) {
	opponents := gi.State.GetOtherTeam(m.team)
	enemyGoal := gi.EnemyGoalCenter(m.team)
	ownGoalCenter := m.ownGoalCenterFromEnemyGoal(enemyGoal)

	bestThreat := -1.0
	var bestOpp Position
	found := false

	var i ID
	for i = 0; i < TEAM_SIZE; i++ {
		oppPos, err := opponents[i].GetPosition()
		if err != nil {
			continue
		}
		if oppPos.Dist2d(ballPos) < 500.0 {
			continue
		}

		goalDist := oppPos.Dist2d(ownGoalCenter)
		passDist := math.Max(oppPos.Dist2d(ballPos), 1.0)

		threat := (7000.0-goalDist)/passDist + 0.001*math.Abs(oppPos.Y)
		if threat > bestThreat {
			bestThreat = threat
			bestOpp = oppPos
			found = true
		}
	}

	if !found {
		return Position{}, false
	}

	vec := bestOpp.Sub(&ballPos)
	vecScaled := vec.Scale(passLaneBlockDepth)
	block := ballPos.Add(&vecScaled)
	return block, true
}

func spreadTargets(targets map[info.ID]Position, minSep float64) map[info.ID]Position {
	adjusted := make(map[info.ID]Position, len(targets))
	for id, p := range targets {
		adjusted[id] = p
	}

	ids := make([]info.ID, 0, len(adjusted))
	for id := range adjusted {
		ids = append(ids, id)
	}

	for iter := 0; iter < 3; iter++ {
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				id1 := ids[i]
				id2 := ids[j]
				p1 := adjusted[id1]
				p2 := adjusted[id2]

				dist := p1.Dist2d(p2)
				if dist >= minSep || dist < 1e-6 {
					continue
				}

				pushDir := p2.Sub(&p1).Normalize2d()
				push := pushDir.Scale((minSep - dist) / 2.0)

				negPush := push.Scale(-1)
				p1 = p1.Add(&negPush)
				p2 = p2.Add(&push)

				adjusted[id1] = p1
				adjusted[id2] = p2
			}
		}
	}

	return adjusted
}

type candidateRole int

const (
	candPress candidateRole = iota
	candCentral
	candWideLeft
	candWideRight
	candSupport
)

type roleCandidate struct {
	botID   info.ID
	role    candidateRole
	score   float64
	target  Position
	yOffset float64
}

func (m *TwoBotDefence) buildRoleTargets(ballPos, ownGoalCenter Position, gi *GameInfo) map[candidateRole]Position {
	return map[candidateRole]Position{
		candCentral:   m.calculateCentralCoverPos(ballPos, ownGoalCenter),
		candPress:     m.calculatePressPos(ballPos),
		candWideLeft:  m.calculateWidePos(ballPos, ownGoalCenter, +1),
		candWideRight: m.calculateWidePos(ballPos, ownGoalCenter, -1),
		candSupport:   m.calculateSupportPos(ballPos, ownGoalCenter, gi),
	}
}

func mapCandidateToRole(c candidateRole) (roles.RoleType, float64) {
	switch c {
	case candPress:
		return roles.RolePress, 0
	case candCentral:
		return roles.RoleCentral, 0
	case candWideLeft:
		return roles.RoleWide, +900
	case candWideRight:
		return roles.RoleWide, -900
	default:
		return roles.RoleSupport, 0
	}
}

func (m *TwoBotDefence) assignRoles(
	gi *GameInfo,
	defenders map[info.ID]*roles.DefenseRole,
	ballPos Position,
	ownGoalCenter Position,
) map[info.ID]Position {
	targetsByRole := m.buildRoleTargets(ballPos, ownGoalCenter, gi)
	candidates := []roleCandidate{}

	for _, id := range m.activeBots {
		botPos, err := gi.State.GetRobotPosition(m.team, id)
		if err != nil {
			continue
		}

		for roleKey, target := range targetsByRole {
			score := botPos.Dist2d(target)

			currentRole, currentOffset := defenders[id].GetRole()
			mappedRole, mappedOffset := mapCandidateToRole(roleKey)
			if currentRole == mappedRole && math.Abs(currentOffset-mappedOffset) < 1e-3 {
				score -= roleStickBonus
			}

			if roleKey == candCentral {
				score -= 120.0
			}

			candidates = append(candidates, roleCandidate{
				botID:   id,
				role:    roleKey,
				score:   score,
				target:  target,
				yOffset: mappedOffset,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score < candidates[j].score
	})

	assignedBots := map[info.ID]bool{}
	filledRoles := map[candidateRole]bool{}
	finalTargets := map[info.ID]Position{}

	priorityRoles := []candidateRole{candCentral, candPress, candWideLeft, candWideRight, candSupport}

	for _, wantedRole := range priorityRoles {
		bestIdx := -1
		bestScore := math.Inf(1)

		for i, c := range candidates {
			if c.role != wantedRole {
				continue
			}
			if assignedBots[c.botID] {
				continue
			}
			if c.score < bestScore {
				bestScore = c.score
				bestIdx = i
			}
		}

		if bestIdx == -1 {
			continue
		}

		chosen := candidates[bestIdx]
		roleType, yOffset := mapCandidateToRole(chosen.role)
		defenders[chosen.botID].SetRole(roleType, yOffset)
		assignedBots[chosen.botID] = true
		filledRoles[chosen.role] = true
		finalTargets[chosen.botID] = chosen.target
	}

	for _, id := range m.activeBots {
		if assignedBots[id] {
			continue
		}
		supportTarget := targetsByRole[candSupport]
		defenders[id].SetRole(roles.RoleSupport, 0)
		finalTargets[id] = supportTarget
	}

	finalTargets = spreadTargets(finalTargets, minDefenderSeparation)

	for id, pos := range finalTargets {
		defenders[id].SetAssignedTarget(pos)
	}

	_ = filledRoles
	return finalTargets
}

func (m *TwoBotDefence) run() {
	defenders := make(map[info.ID]*roles.DefenseRole)

	gi := <-m.incomingGameInfo

	for _, id := range m.activeBots {
		defender := roles.NewDefenseRole(id, m.ActivityHandler, &gi, m.team)
		defender.Init()
		defenders[id] = defender
	}

	var isNear bool

	for {
		gi = <-m.incomingGameInfo

		enemyGoal := gi.EnemyGoalCenter(m.team)
		ownGoalCenter := m.ownGoalCenterFromEnemyGoal(enemyGoal)
		sign := math.Copysign(1.0, ownGoalCenter.X)

		ballPos, _ := gi.State.GetBall().GetEstimatedPosition()

		m.assignRoles(&gi, defenders, ballPos, ownGoalCenter)

		attackerPos, found := m.getAttackerPosition(&gi)
		if found {
			attackerSignedX := attackerPos.X * sign
			if !isNear && attackerSignedX > 2800 {
				isNear = true
			} else if isNear && attackerSignedX < 2400 {
				isNear = false
			}
		}

		if isNear && found && len(m.activeBots) >= 1 {
			type botDist struct {
				id   info.ID
				dist float64
			}
			sorted := make([]botDist, 0, len(m.activeBots))
			for _, id := range m.activeBots {
				pos, err := gi.State.GetRobotPosition(m.team, id)
				if err != nil {
					sorted = append(sorted, botDist{id, math.Inf(1)})
					continue
				}
				sorted = append(sorted, botDist{id, pos.Dist2d(attackerPos)})
			}
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].dist < sorted[j].dist
			})

			numWall := 2
			if len(sorted) < numWall {
				numWall = len(sorted)
			}
			wallPositions := m.calcWallPositions(attackerPos, ownGoalCenter, numWall)
			for i := 0; i < numWall; i++ {
				id := sorted[i].id
				defenders[id].SetWallPosition(wallPositions[i])
				defenders[id].TriggerEvent("ATTACKER_NEAR")
			}

			for i := numWall; i < len(sorted); i++ {
				defenders[sorted[i].id].TriggerEvent("ATTACKER_FAR")
			}
		} else {
			for _, id := range m.activeBots {
				defenders[id].TriggerEvent("ATTACKER_FAR")
			}
		}

		for _, id := range m.activeBots {
			defenders[id].Run()
		}

		time.Sleep(1 * time.Millisecond)
	}
}