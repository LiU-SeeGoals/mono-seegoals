package ai

import (
	"math"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/helper"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

type CombinedPlan struct {
	plannerCore
}

func NewCombinedPlan(team Team) *CombinedPlan {
	return &CombinedPlan{
		plannerCore: plannerCore{team: team},
	}
}

func (m *CombinedPlan) Init(
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

// ---------------------------------------------------------------------------
// Offense helpers (ported from pass_scenario.go / GameScenario)
// ---------------------------------------------------------------------------

// getRobotClosestToPosition returns the robot from activeRobots that is
// closest to the given target position.
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

// getRobotForBall picks the offensive robot that should pursue the ball,
// using a short velocity-based lookahead so the robot intercepts rather
// than chases.
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

// ---------------------------------------------------------------------------
// Defense helpers (ported from plan_2bot_defence.go / TwoBotDefence)
// ---------------------------------------------------------------------------

// getAttackerPosition finds the yellow robot closest to the ball — i.e.
// the most likely attacker we need to defend against.
func (m *CombinedPlan) getAttackerPosition(gi *GameInfo) (Position, bool) {
	yellowTeam := gi.State.GetTeam(Yellow)
	ballPos, _ := gi.State.GetBall().GetEstimatedPosition()

	minDist := math.Inf(1)
	var closestPos Position
	found := false

	var i ID
	for i = 0; i < TEAM_SIZE; i++ {
		robotPos, err := yellowTeam[i].GetPosition()
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

// calcWallPositions returns wall positions for three defensive bots.
// The bots are placed perpendicular to the shot line, centered at the
// midpoint between attacker and goal, spaced wallSpacing apart.
func (m *CombinedPlan) calcWallPositions(attackerPos Position) (Position, Position, Position) {
	// Shot vector: attacker → goal, normalized
	shotVec := blueGoalCenter.Sub(&attackerPos)
	shotVecNorm := shotVec.Normalize2d()

	// Midpoint between attacker and goal — wall forms here
	mid := attackerPos.Add(&blueGoalCenter)
	mid.Div2d(2.0)

	// Perpendicular to shot line (rotate 90°)
	perpVec := Position{X: -shotVecNorm.Y, Y: shotVecNorm.X, Z: 0, Angle: 0}

	// Three bots: center, +wallSpacing, -wallSpacing
	offset := perpVec.Scale(wallSpacing)
	bot1Pos := mid.Sub(&offset) // one side
	bot2Pos := mid              // center
	bot3Pos := mid.Add(&offset) // other side

	// All bots face the attacker
	bot1Pos.Angle = bot1Pos.AngleToPosition(attackerPos)
	bot2Pos.Angle = bot2Pos.AngleToPosition(attackerPos)
	bot3Pos.Angle = bot3Pos.AngleToPosition(attackerPos)

	return bot1Pos, bot2Pos, bot3Pos
}

// ---------------------------------------------------------------------------
// Main loop — combines offense + defense in a single goroutine
// ---------------------------------------------------------------------------

func (m *CombinedPlan) run() {
	// Partition robots: 1-3 offense, 4-6 defense
	offenseRobots := []info.ID{1, 2, 3}
	defenseRobots := []info.ID{4, 5, 6}

	gi := <-m.incomingGameInfo

	// Initialize offense roles (from pass_scenario logic)
	kickers := make(map[info.ID]*roles.OffenseRole)
	for _, id := range offenseRobots {
		kicker := roles.NewOffenseRole(id, m.ActivityHandler, &gi, m.team)
		kicker.Init()
		kickers[id] = kicker
	}

	// Initialize defense roles (from plan_2bot_defence logic)
	defenders := make(map[info.ID]*roles.DefenseRole)
	for _, id := range defenseRobots {
		defender := roles.NewDefenseRole(id, m.ActivityHandler, &gi, m.team)
		defender.Init()
		defenders[id] = defender
	}

	for {
		tickStart := time.Now()
		gi = <-m.incomingGameInfo

		// === OFFENSE: closest robot chases ball, others support ===
		closestId := m.getRobotForBall(&gi, offenseRobots)
		for _, id := range offenseRobots {
			if id != closestId {
				kickers[id].TriggerEvent("BALL_LOST")
			}
		}
		kickers[closestId].TriggerEvent("BALL_OWNER")
		for _, kicker := range kickers {
			kicker.Run()
		}

		// === DEFENSE: wall formation when attacker threatens ===
		attackerPos, found := m.getAttackerPosition(&gi)
		if found && attackerPos.X > attackerThreatX {
			// Attacker is in our half — form the 3-bot wall
			bot4Pos, bot5Pos, bot6Pos := m.calcWallPositions(attackerPos)
			defenders[4].SetWallPosition(bot4Pos)
			defenders[5].SetWallPosition(bot5Pos)
			defenders[6].SetWallPosition(bot6Pos)
			for _, d := range defenders {
				d.TriggerEvent("ATTACKER_NEAR")
			}
		} else {
			// Attacker is in their half — press and cover
			for _, d := range defenders {
				d.TriggerEvent("ATTACKER_FAR")
			}
		}
		for _, d := range defenders {
			d.Run()
		}

		helper.PaceLoop(tickStart, helper.PlannerLoopPeriod, "combined_plan")
	}
}
