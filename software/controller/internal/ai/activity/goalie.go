package ai

import (
	"fmt"
	"math"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

const (
	goalieForwardPenaltyAreaFraction = 0.5
	BALL_JITTER_DISTANCE             = 1.0
	SHOT_THREAT_DISTANCE             = 350.0
)

type goalieMovementBounds struct {
	frontX        float64
	backX         float64
	goalHalfWidth float64
}

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
	bounds, ok := newGoalieMovementBounds(gi, g.team)
	if !ok {
		return &action.Stop{Id: int(g.id)}
	}

	ball := gi.State.GetBall()

	// Current ball position
	ballPos, err := ball.GetEstimatedPosition()
	if err != nil {
		fmt.Println("Error getting ball position:", err)
		return NewMoveToPosition(g.team, g.id, info.Position{X: 0, Y: 0}).GetAction(gi)
	}

	// Follow the ball in X, but never leave the allowed goalie X interval.
	targetX := ballPos.X
	minX := math.Min(bounds.frontX, bounds.backX)
	maxX := math.Max(bounds.frontX, bounds.backX)
	if targetX < minX {
		targetX = minX
	} else if targetX > maxX {
		targetX = maxX
	}
	goalieX := targetX
	goalSize := bounds.goalHalfWidth
	goalieY := ballPos.Y

	// If an opponent has the ball, or is still close enough to be the likely shooter,
	// predict where they are aiming on our goal line.
	if shooter := threateningOpponent(gi, g.team, ballPos); shooter != nil {
		if oppPos, err := shooter.GetPosition(); err == nil {
			if yHit, ok := predictShotY(oppPos, goalieX, goalSize, ballPos.Y); ok {
				goalieY = yHit
			}
		}
	} else if yHit, ok := predictBallPathY(ball, goalieX, goalSize); ok {
		// Otherwise, predict the ball trajectory when it is already moving toward our goal.
		goalieY = yHit
	}

	if goalieY > goalSize {
		goalieY = goalSize
	} else if goalieY < -goalSize {
		goalieY = -goalSize
	}

	goaliePos := info.Position{X: goalieX, Y: goalieY, Z: 0.0, Angle: 0.0}

	myRobotPos, _ := gi.State.GetTeam(g.team)[g.id].GetPosition()
	goaliePos.Angle = myRobotPos.AngleToPosition(ballPos)

	move := NewMoveToPosition(g.team, g.id, goaliePos)
	// The keeper must be able to enter its defense area. Disabling goal-line
	// obstacles still leaves robot and restart-ball obstacles enabled in RRT.
	move.AvoidGoallines(false)
	move.SetUseRRT(goalieShouldUseRRT(gi))
	return move.GetAction(gi)
}

func newGoalieMovementBounds(gi *info.GameInfo, team info.Team) (goalieMovementBounds, bool) {
	if gi == nil {
		return goalieMovementBounds{}, false
	}

	geometry, ok := gi.FieldGeometry()
	if !ok || geometry.GoalWidth <= 0 || geometry.GoalDepth <= 0 {
		return goalieMovementBounds{}, false
	}

	goalSign := gi.OwnHalfXSign(team)
	goalLineX := goalSign * geometry.Length / 2
	forwardInset := geometry.PenaltyAreaDepth * goalieForwardPenaltyAreaFraction
	if forwardInset <= 0 {
		forwardInset = geometry.GoalDepth
	}

	return goalieMovementBounds{
		frontX:        goalLineX - goalSign*forwardInset,
		backX:         goalLineX - goalSign*geometry.GoalDepth,
		goalHalfWidth: geometry.GoalWidth / 2,
	}, true
}

func goalieShouldUseRRT(gi *info.GameInfo) bool {
	if gi == nil || gi.Status == nil {
		return false
	}
	gameEvent := gi.Status.GetGameEvent()
	return gameEvent != nil && !gameEvent.BallInPlay
}

// Achieved returns whether this action is "complete".
// The goalie never really finishes, so we return false unless higher-level AI changes it.
func (g *Goalie) Achieved(*info.GameInfo) bool {
	return false
}

func (m *Goalie) GetID() info.ID {
	return m.id
}

//Predict the opponent while oppponent is close enough to the ball
//Returns the opponent that should be treated as the ball possessor
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

// predictBallPathY estimates where the moving ball crosses the goalie's X line.
// It ignores tiny displacements to avoid reacting to jitter, and only returns true
// when the current ball path intersects the goal mouth.
func predictBallPathY(ball *info.Ball, goalieX float64, goalSize float64) (float64, bool) {
	currentPos, currentTime, previousPos, previousTime, err := ball.GetLatestTwoPositionsTime()
	if err != nil {
		return 0, false
	}

	dt := float64(currentTime - previousTime)
	if dt <= 0 {
		return 0, false
	}

	dx := currentPos.X - previousPos.X
	dy := currentPos.Y - previousPos.Y
	displacement := math.Hypot(dx, dy)
	if displacement < BALL_JITTER_DISTANCE {
		return 0, false
	}

	if math.Abs(dx) < 1e-6 {
		return 0, false
	}

	velocityX := dx / dt
	if math.Abs(velocityX) < 1e-6 {
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
