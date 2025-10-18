package ai

import (
	"fmt"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

// Constants for goalie positioning
const (
	// Goalie position constraints - these will be adjusted based on team half
	GOALIE_LINE_WIDTH = 1000 // Width of the goalie's movement range (500 to each side)
	// GOALIE_DIST_FROM_CENTER = 5500 // Distance from center to goalie line
	GOALIE_DIST_FROM_CENTER = 4300                           // Distance from center to goalie line
	GOAL_BEHIND_DIST        = GOALIE_DIST_FROM_CENTER + 1200 // Distance from center to position behind the goal
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

// calculateInterceptionPoint determines where the goalie should position itself
// based on the ball position, the "behind goal" reference point, and which half we're defending
func (g *Goalie) calculateInterceptionPoint(ballPos info.Position, isPositiveHalf bool) info.Position {
	// Determine goalie line X position and behind-goal X position based on which half we're defending
	var goalieX, goalBehindX float64
	var xMultiplier float64

	if isPositiveHalf {
		// We're defending the positive X side
		xMultiplier = 1.0
	} else {
		// We're defending the negative X side
		xMultiplier = -1.0
	}

	// Calculate actual coordinates based on which half we're defending
	goalieX = xMultiplier * GOALIE_DIST_FROM_CENTER
	goalBehindX = xMultiplier * GOAL_BEHIND_DIST
	maxBottomY := float64(-GOALIE_LINE_WIDTH / 2)
	maxTopY := float64(GOALIE_LINE_WIDTH / 2)

	// If the ball is not detected or has invalid position, return center position
	if ballPos.X == 0 && ballPos.Y == 0 && ballPos.Z == 0 {
		return info.Position{X: goalieX, Y: 0, Z: 0, Angle: 0}
	}

	// Handle case where ball is directly in line with goal (to avoid division by zero)
	if ballPos.Y == 0 {
		return info.Position{X: goalieX, Y: 0, Z: 0, Angle: 0}
	}

	// Calculate slope of the line from ball to behind-goal position
	// Formula: slope = (y2 - y1) / (x2 - x1)
	// Where (x1,y1) is the ball position and (x2,y2) is the behind-goal position (goalBehindX, 0)
	slope := (0 - ballPos.Y) / (goalBehindX - ballPos.X)

	// Calculate the y-coordinate where the line intersects the goalie's movement line
	// Using the point-slope formula: y - y1 = m(x - x1)
	// Solving for y when x = goalieX
	interceptY := ballPos.Y + slope*(goalieX-ballPos.X)

	// For very small GOAL_

	// If the result seems to be mirrored, negate the Y value to correct the direction
	// This will make sure if the ball is on the left, the goalie moves left
	interceptY = -interceptY

	// Constrain the position to the max top/bottom boundaries
	if interceptY < maxBottomY {
		interceptY = maxBottomY
	} else if interceptY > maxTopY {
		interceptY = maxTopY
	}

	// Return the goalie position
	return info.Position{X: goalieX, Y: -interceptY, Z: 0, Angle: 0}
}

func (g *Goalie) GetAction(gi *info.GameInfo) action.Action {
	ball := gi.State.GetBall()

	// Current ball position
	ballPos, err := ball.GetEstimatedPosition()
	if err != nil {
		fmt.Println("Error getting ball position:", err)
		return NewMoveToPosition(g.team, g.id, info.Position{X: 0, Y: 0}).GetAction(gi)
	}

	// Determine which half we're defending
	isBlueTeam := g.team == info.Blue
	isBlueOnPositiveHalf := gi.Status.GetBlueTeamOnPositiveHalf()
	isDefendingPositiveHalf := (isBlueTeam && isBlueOnPositiveHalf) || (!isBlueTeam && !isBlueOnPositiveHalf)

	// Goalie line X position
	xMultiplier := 1.0
	if !isDefendingPositiveHalf {
		xMultiplier = -1.0
	}
	goalieX := xMultiplier * GOALIE_DIST_FROM_CENTER

	// Default: follow standard positioning
	goaliePos := g.calculateInterceptionPoint(ballPos, isDefendingPositiveHalf)

	// If ball is free and moving toward our goal — try to intercept its path

	ballVel := ball.GetVelocity()

	// Check if moving toward our goal
	movingTowardGoal := (xMultiplier > 0 && ballVel.X > 0) || (xMultiplier < 0 && ballVel.X < 0)
	if movingTowardGoal && ballVel.X != 0 {
		// Predict where the ball will cross the goalie's X line
		tHit := (goalieX - ballPos.X) / ballVel.X
		if tHit > 0 { // only if in the future
			predictedY := ballPos.Y + ballVel.Y*tHit

			// Clamp within goalie range
			if predictedY < -GOALIE_LINE_WIDTH/2 {
				predictedY = -GOALIE_LINE_WIDTH / 2
			} else if predictedY > GOALIE_LINE_WIDTH/2 {
				predictedY = GOALIE_LINE_WIDTH / 2
			}

			// Set predicted interception point
			goaliePos = info.Position{X: goalieX, Y: predictedY, Z: 0, Angle: 0}
		}
	}

	// Move to calculated position
	move := NewMoveToPosition(g.team, g.id, goaliePos)
	return move.GetAction(gi)
}

// Achieved returns whether this action is "complete".
// The goalie never really finishes, so we return false unless higher-level AI changes it.
func (g *Goalie) Achieved(*info.GameInfo) bool {
	return false
}

func (m *Goalie) GetID() info.ID {
	return m.id
}
