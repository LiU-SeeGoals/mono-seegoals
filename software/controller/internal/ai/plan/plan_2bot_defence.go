package ai

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

const wallSpacing = 200.0 // mm — spacing between adjacent wall bots

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
	activities *[TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team Team,
) {
	m.incomingGameInfo = incoming
	m.ActivityHandler.Activities = activities
	m.ActivityHandler.Activity_lock = lock
	m.team = team

	go m.run()
}

// getAttackerPosition finds the opponent robot closest to the ball.
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

// calcWallPositions returns n positions perpendicular to the shot line,
// centered at the midpoint between attacker and own goal.
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

func (m *TwoBotDefence) run() {
	defenders := make(map[info.ID]*roles.DefenseRole)

	gi := <-m.incomingGameInfo

	for _, id := range m.activeBots {
		defender := roles.NewDefenseRole(id, m.ActivityHandler, &gi, m.team)
		defender.Init()
		defenders[id] = defender
	}

	// isNear: attacker entered our defensive half — engage wall mode.
	// Hysteresis: enter at 2800mm, exit at 2400mm.
	var isNear bool

	for {
		gi = <-m.incomingGameInfo
		enemyGoal := gi.EnemyGoalCenter(m.team)
		ownGoalCenter := Position{X: -enemyGoal.X, Y: 0, Z: 0, Angle: 0}
		sign := math.Copysign(1.0, -enemyGoal.X)

		ballPos, _ := gi.State.GetBall().GetEstimatedPosition()

		
		type defDist struct {
			id   info.ID
			dist float64
		}
		sorted := make([]defDist, 0, len(m.activeBots))
		for _, id := range m.activeBots {
			pos, err := gi.State.GetRobotPosition(m.team, id)
			if err != nil {
				sorted = append(sorted, defDist{id, math.Inf(1)})
				continue
			}
			sorted = append(sorted, defDist{id, ballPos.Dist2d(pos)})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].dist < sorted[j].dist
		})

		type roleAssignment struct {
			role    roles.RoleType
			yOffset float64
		}
		roleOrder := []roleAssignment{
			{roles.RolePress, 0},
			{roles.RoleCentral, 0},
			{roles.RoleWide, +600},
			{roles.RoleWide, -600},
			{roles.RoleSupport, 0},
		}
		for i, d := range sorted {
			ra := roleAssignment{roles.RoleSupport, 0}
			if i < len(roleOrder) {
				ra = roleOrder[i]
			}
			defenders[d.id].SetRole(ra.role, ra.yOffset)
		}

		// Attacker threat detection with hysteresis
		attackerPos, found := m.getAttackerPosition(&gi)
		if found {
			attackerSignedX := attackerPos.X * sign
			if !isNear && attackerSignedX > 2800 {
				isNear = true
			} else if isNear && attackerSignedX < 2400 {
				isNear = false
			}
		}

		if isNear && found && len(sorted) >= 1 {
			
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
			// Support bots hold formation, do not join the wall
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

		time.Sleep(time.Millisecond * 1)
	}
}
