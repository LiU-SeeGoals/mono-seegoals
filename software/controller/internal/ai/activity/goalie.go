package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	SHOT_THREAT_DISTANCE   = 350.0
	TRACKED_BALL_MIN_SPEED = 0.05
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
	myRobotPos, err := gi.State.GetTeam(g.team)[g.id].GetPosition()
	if err != nil {
		fmt.Println("Error getting goalie position:", err)
		return NewMoveToPosition(g.team, g.id, info.Position{X: 0, Y: 0}).GetAction(gi)
	}

	// Current ball position
	ballPos, err := ball.GetEstimatedPosition()
	if err != nil {
		fmt.Println("Error getting ball position:", err)
		return NewMoveToPosition(g.team, g.id, info.Position{X: 0, Y: 0}).GetAction(gi)
	}

	goalLineX, defendLimitX, goalSize, ok := goalieFieldBounds(gi, myRobotPos)
	if !ok {
		fmt.Println("goalie: missing field geometry, falling back to origin")
		return NewMoveToPosition(g.team, g.id, info.Position{X: 0, Y: 0}).GetAction(gi)
	}

	// Follow the ball in X, but never leave the own goal-to-penalty interval.
	targetX := ballPos.X
	minX := math.Min(goalLineX, defendLimitX)
	maxX := math.Max(goalLineX, defendLimitX)
	if targetX < minX {
		targetX = minX
	} else if targetX > maxX {
		targetX = maxX
	}
	goalieX := targetX
	goalieY := ballPos.Y

	// If an opponent has the ball, or is still close enough to be the likely shooter,
	// predict where they are aiming on our goal line.
	if shooter := threateningOpponent(gi, g.team, ballPos); shooter != nil {
		if oppPos, err := shooter.GetPosition(); err == nil {
			if yHit, ok := predictShotY(oppPos, goalieX, goalSize, ballPos.Y); ok {
				goalieY = yHit
			}
		}
	} else if yHit, ok := predictBallPathY(gi, ballPos, goalieX, goalSize); ok {
		// Otherwise, predict the ball trajectory when it is already moving toward our goal.
		goalieY = yHit
	}

	if goalieY > goalSize {
		goalieY = goalSize
	} else if goalieY < -goalSize {
		goalieY = -goalSize
	}

	goaliePos := info.Position{X: goalieX, Y: goalieY, Z: 0.0, Angle: 0.0}

	lookAtBall := myRobotPos.AngleToPosition(ballPos)

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

func goalieFieldBounds(gi *info.GameInfo, goaliePos info.Position) (goalLineX float64, defendLimitX float64, goalWidth float64, ok bool) {
	leftGoalLine := gi.GetFieldLine("LeftGoalLine")
	rightGoalLine := gi.GetFieldLine("RightGoalLine")
	leftPenaltyLine := gi.GetFieldLine("LeftPenaltyStretch")
	rightPenaltyLine := gi.GetFieldLine("RightPenaltyStretch")
	if leftGoalLine == nil || rightGoalLine == nil || leftPenaltyLine == nil || rightPenaltyLine == nil {
		return 0, 0, 0, false
	}

	leftGoalX := float64(leftGoalLine.GetP1().GetX())
	rightGoalX := float64(rightGoalLine.GetP1().GetX())
	leftPenaltyX := float64(leftPenaltyLine.GetP1().GetX())
	rightPenaltyX := float64(rightPenaltyLine.GetP1().GetX())

	if math.Abs(goaliePos.X-leftGoalX) <= math.Abs(goaliePos.X-rightGoalX) {
		goalLineX = leftGoalX
		defendLimitX = leftPenaltyX
	} else {
		goalLineX = rightGoalX
		defendLimitX = rightPenaltyX
	}

	goalWidth = gi.GoalWidth()
	if goalWidth <= 0 {
		return 0, 0, 0, false
	}

	return goalLineX, defendLimitX, goalWidth, true
}

// Predict the opponent while oppponent is close enough to the ball.
// Returns the opponent that should be treated as the ball possessor.
func threateningOpponent(gi *info.GameInfo, team info.Team, ballPos info.Position) *info.Robot {
	ball := gi.State.GetBall()
	if poss := ball.GetPossessor(); poss != nil && poss.GetTeam() != team {
		return poss
	}

	opponents := gi.State.GetOtherTeam(team)
	var best *info.Robot
	bestDist := math.Inf(1)

	for _, robot := range opponents {
		if robot == nil || !robot.IsActive() {
			continue
		}

		dist := ballPos.Distance(robot.DribblerPos())
		if dist > SHOT_THREAT_DISTANCE || dist >= bestDist {
			continue
		}

		best = robot
		bestDist = dist
	}

	return best
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

// predictBallPathY estimates where the tracked ball velocity crosses the goalie's X line.
// It uses tracked velocity for faster reaction and only returns true when the current
// ball path intersects the goal mouth.
func predictBallPathY(gi *info.GameInfo, fallbackPos info.Position, goalieX float64, goalSize float64) (float64, bool) {
	trackedBall := gi.State.GetTrackedBall()
	ballVel, ok := trackedBall.GetTrackedVelocity()
	if !ok {
		return 0, false
	}

	speed := ballVel.Norm2d()
	if speed < TRACKED_BALL_MIN_SPEED {
		return 0, false
	}

	currentPos, posOK := trackedBall.GetTrackedPosition()
	if !posOK {
		currentPos = fallbackPos
	}

	dx := ballVel.X
	dy := ballVel.Y
	if math.Abs(dx) < 1e-6 {
		return 0, false
	}

	t := (goalieX - currentPos.X) / dx
	if t < 0 {
		return 0, false
	}

	yHit := currentPos.Y + t*dy
	if yHit < -goalSize || yHit > goalSize {
		return 0, false
	}

	return yHit, true
}
