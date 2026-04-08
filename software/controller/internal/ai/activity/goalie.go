package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

// Constants for goalie positioning
const (
	// Goalie position constraints - these will be adjusted based on team half
	GOALIE_LINE_WIDTH = 1000 // Width of the goalie's movement range (500 to each side)
	// GOALIE_DIST_FROM_CENTER = 5500 // Distance from center to goalie line
	GOALIE_DIST_FROM_CENTER = 3500 // Distance from center to goalie line
	GOAL_BEHIND_DIST        = 4300 // Distance from center to position behind the goal
)

type Goalie struct {
	GenericComposition
	team info.Team
	id   info.ID
	Activity
}

func (g *Goalie) String() string {
	return fmt.Sprintf("Goalie(%d, %d)", g.team, g.id)
}

// NewGoalie creates a new Goalie struct.
func NewGoalie(team info.Team, id info.ID) *Goalie {
	return &Goalie{
		GenericComposition: GenericComposition{
			team: team,
			id:   id,
		},
		team: team,
		id:   id,
	}
}

func (g *Goalie) GetAction(gi *info.GameInfo) action.Action {
	ball := gi.State.GetBall()

	// Current ball position
	ballPos, err := ball.GetEstimatedPosition()
	if err != nil {
		fmt.Println("Error getting ball position:", err)
		return NewMoveToPosition(g.team, g.id, info.Position{X: 0, Y: 0}).GetAction(gi)
	}
	printOpponentNearBall(gi, g.team, ballPos)

	// Determine which half we're defending
	isBlueTeam := g.team == info.Blue
	isBlueOnPositiveHalf := gi.Status.GetBlueTeamOnPositiveHalf()
	isDefendingPositiveHalf := (isBlueTeam && isBlueOnPositiveHalf) || (!isBlueTeam && !isBlueOnPositiveHalf)

	xMultiplier := -1.0
	if !isDefendingPositiveHalf {
		xMultiplier = 1.0
	}
	// Let the goalie move in both X and Y: mirror the ball Y but also slide along X
	// between the regular goalie line and a deeper "behind goal" line.
	absBallX := math.Abs(ballPos.X)
	clampedX := math.Max(GOALIE_DIST_FROM_CENTER, math.Min(GOAL_BEHIND_DIST, absBallX))
	// Default goalieX on the line, but try to place between ball and goal within limits
	goalieX := xMultiplier * clampedX
	if poss := ball.GetPossessor(); poss != nil && poss.GetTeam() != g.team {
		// Place goalie between goal line and the ball X, clamped to limits
		goalLineX := xMultiplier * GOALIE_DIST_FROM_CENTER
		behindLimitX := xMultiplier * GOAL_BEHIND_DIST
		targetX := (goalLineX + ballPos.X) / 2.0
		minX := math.Min(goalLineX, behindLimitX)
		maxX := math.Max(goalLineX, behindLimitX)
		if targetX < minX {
			targetX = minX
		} else if targetX > maxX {
			targetX = maxX
		}
		goalieX = targetX
	}
	goalSize := 1110.0
	goalieY := ballPos.Y

	// If an opponent has the ball, predict where they are aiming on our goal line.
	if poss := ball.GetPossessor(); poss != nil && poss.GetTeam() != g.team {
		if oppPos, err := poss.GetPosition(); err == nil {
			if yHit, ok := predictShotY(oppPos, goalieX, goalSize, ballPos.Y); ok {
				goalieY = yHit
			}
		}
	}

	if goalieY > goalSize {
		goalieY = goalSize
	} else if goalieY < -goalSize {
		goalieY = -goalSize
	}

	goaliePos := info.Position{X: goalieX, Y: goalieY, Z: 0.0, Angle: 0.0}

	myRobotPos, _ := gi.State.GetTeam(g.team)[g.id].GetPosition()
	lookAtBall := myRobotPos.AngleToPosition(ballPos)

	// If we have the ball, try to pass to the closest teammate (excluding self).
	// If we have the ball, try a simple straight kick toward opponent goal
	if ball.GetPossessor() == gi.State.GetTeam(g.team)[g.id] {
		target := nearestTeammatePos(gi, g.team, g.id)
		if target == nil {
			targetX := -goalieX // fallback: opponent goal roughly opposite our own
			t := info.Position{X: targetX, Y: 0, Z: 0, Angle: 0}
			return NewKickBall(g.team, g.id, t, ballPos).GetAction(gi)
		}
		return NewKickBall(g.team, g.id, *target, ballPos).GetAction(gi)
	}

	//move := NewMoveToPosition(g.team, g.id, goaliePos)
	act := action.MoveTo{}
	act.Id = int(g.id)
	act.Team = g.team
	act.Pos = myRobotPos
	act.Dest = goaliePos
	act.Dest.Angle = lookAtBall
	act.Dribble = false

	return &act

	//return move.GetAction(gi)
}

// Achieved returns whether this action is "complete".
// The goalie never really finishes, so we return false unless higher-level AI changes it.
func (g *Goalie) Achieved(*info.GameInfo) bool {
	return false
}

func (m *Goalie) GetID() info.ID {
	return m.id
}

// predictShotY estimates where the opponent's aim line hits our goal line.
// Returns yHit and a boolean indicating if the prediction was valid.
func predictShotY(opponent info.Position, goalieX float64, goalSize float64, fallbackY float64) (float64, bool) {
	x0, y0, theta := opponent.X, opponent.Y, opponent.Angle
	dx := math.Cos(theta)
	dy := math.Sin(theta)
	if math.Abs(dx) < 1e-6 {
		return fallbackY, false
	}

	t := (goalieX - x0) / dx
	if t < 0 {
		return fallbackY, false
	}

	yHit := y0 + t*dy
	if yHit > goalSize {
		yHit = goalSize
	} else if yHit < -goalSize {
		yHit = -goalSize
	}
	return yHit, true
}

// printOpponentNearBall logs the first opponent found within 300mm of the ball.
func printOpponentNearBall(gi *info.GameInfo, team info.Team, ballPos info.Position) {
	opponents := gi.State.GetOtherTeam(team)
	for id, r := range opponents {
		if r == nil {
			continue
		}
		pos, err := r.GetPosition()
		if err != nil {
			continue
		}
		if pos.Distance(ballPos) < 300 {
			fmt.Printf("Opponent %d near ball\n", id)
			return
		}
	}
}

// nearestTeammatePos returns the position of the closest active teammate to the ball (excluding self).
func nearestTeammatePos(gi *info.GameInfo, team info.Team, self info.ID) *info.Position {
	ballPos, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return nil
	}
	teamRobots := gi.State.GetTeam(team)
	var best *info.Position
	bestDist := math.Inf(1)
	for id, r := range teamRobots {
		if info.ID(id) == self || r == nil {
			continue
		}
		pos, err := r.GetPosition()
		if err != nil {
			continue
		}
		d := pos.Distance(ballPos)
		if d < bestDist {
			bestDist = d
			cp := pos
			best = &cp
		}
	}
	return best
}
