package ai

import (
	"math"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

const (
	defenseThreatGoalWidthFraction = 0.7
	wallSpacing                    = 200.0 // mm — spacing between adjacent wall bots
)

type TwoBotDefence struct {
	plannerCore
}

func NewTwoBotDefence(team info.Team) *TwoBotDefence {
	return &TwoBotDefence{
		plannerCore: plannerCore{
			team: team,
		},
	}
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

// getAttackerPosition finds the opponent closest to the ball.
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

// calcWallPositions returns wall positions for bots 5, 6, and 7.
// The three bots are placed perpendicular to the shot line, centered at the
// midpoint between attacker and goal, spaced wallSpacing apart.
func (m *TwoBotDefence) calcWallPositions(gi *GameInfo, attackerPos Position) (Position, Position, Position, bool) {
	goalCenter, ok := gi.OwnGoalCenter(m.team)
	if !ok {
		return Position{}, Position{}, Position{}, false
	}
	// Shot vector: attacker → goal, normalized
	shotVec := goalCenter.Sub(&attackerPos)
	shotVecNorm := shotVec.Normalize2d()

	// Midpoint between attacker and goal — wall forms here
	mid := attackerPos.Add(&goalCenter)
	mid.Div2d(2.0)

	// Perpendicular to shot line (rotate 90°)
	perpVec := Position{X: -shotVecNorm.Y, Y: shotVecNorm.X, Z: 0, Angle: 0}

	// Three bots: center, +wallSpacing, -wallSpacing
	offset := perpVec.Scale(wallSpacing)
	bot5Pos := mid.Sub(&offset) // one side
	bot6Pos := mid              // center
	bot7Pos := mid.Add(&offset) // other side

	// All bots face the attacker
	bot5Pos.Angle = bot5Pos.AngleToPosition(attackerPos)
	bot6Pos.Angle = bot6Pos.AngleToPosition(attackerPos)
	bot7Pos.Angle = bot7Pos.AngleToPosition(attackerPos)

	return bot5Pos, bot6Pos, bot7Pos, true
}

func (m *TwoBotDefence) attackerInDefensiveZone(gi *GameInfo, attackerPos Position) bool {
	goalCenter, ok := gi.OwnGoalCenter(m.team)
	if !ok {
		return false
	}
	geometry, ok := gi.FieldGeometry()
	if !ok || geometry.PenaltyAreaDepth <= 0 || geometry.GoalWidth <= 0 {
		return false
	}
	xSign := gi.OwnHalfXSign(m.team)
	goalInset := geometry.PenaltyAreaDepth + geometry.GoalWidth*defenseThreatGoalWidthFraction
	boundaryX := goalCenter.X - xSign*goalInset
	return xSign*attackerPos.X > xSign*boundaryX
}

func (m *TwoBotDefence) run() {
	activeRobots := []info.ID{5, 6, 7}
	defenders := make(map[info.ID]*roles.DefenseRole)
	coverMarking := coverMarkingTracker{}

	gi := <-m.incomingGameInfo

	for _, id := range activeRobots {
		defender := roles.NewDefenseRole(id, m.ActivityHandler, &gi, m.team)
		defender.Init()
		defenders[id] = defender
	}

	for {
		gi = <-m.incomingGameInfo
		coverAssignments := coverMarking.update(&gi, m.team, activeRobots)
		for id, defender := range defenders {
			opponentID, assigned := coverAssignments[id]
			defender.SetCoverOpponent(opponentID, assigned)
		}
		setDefenderBallActor(defenders, 0, false)
		owner := observedBallOwner(&gi)
		if owner.valid && owner.team == m.team {
			if _, ok := defenders[owner.id]; ok {
				setDefenderBallActor(defenders, owner.id, true)
			}
		} else if !owner.valid {
			if defenderID, ok := closestReadyDefenderToBall(&gi, m.team, defenders); ok {
				setDefenderBallActor(defenders, defenderID, true)
			}
		}

		attackerPos, found := m.getAttackerPosition(&gi)

		if found && m.attackerInDefensiveZone(&gi, attackerPos) {
			// Attacker is deep in our Vision-reported half — form the 3-bot wall.
			bot5Pos, bot6Pos, bot7Pos, ok := m.calcWallPositions(&gi, attackerPos)
			if !ok {
				continue
			}
			defenders[5].SetWallPosition(bot5Pos)
			defenders[6].SetWallPosition(bot6Pos)
			defenders[7].SetWallPosition(bot7Pos)

			for _, d := range defenders {
				d.TriggerEvent("ATTACKER_NEAR")
			}
		} else {
			// Attacker is in Yellow's half — press and cover
			for _, d := range defenders {
				d.TriggerEvent("ATTACKER_FAR")
			}
		}

		for _, d := range defenders {
			d.Run()
		}

		time.Sleep(time.Millisecond * 1)
	}
}
