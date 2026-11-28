package info

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
)

// Maybe State should also be private, so we can keep track of coordinate system here?
type GameInfo struct {
	State       *GameState
	Status      *GameStatus
	field       *ssl_vision.SSL_GeometryFieldSize
	visionFrame uint64
	clock       Clock
}

// FieldGeometry contains the dimensions reported by SSL-Vision, in millimeters.
type FieldGeometry struct {
	Length           float64
	Width            float64
	GoalWidth        float64
	GoalDepth        float64
	BoundaryWidth    float64
	PenaltyAreaDepth float64
	PenaltyAreaWidth float64
}

func NewGameInfo(capacity int) *GameInfo {
	return &GameInfo{
		State:  NewGameState(capacity),
		Status: NewGameStatus(),
		field:  nil,
		clock:  SystemClock{},
	}
}

// Now is the AI's notion of the current time (the wall clock unless SetClock
// replaced it). Use it instead of time.Now in planners, roles and activities.
func (gi *GameInfo) Now() time.Time {
	if gi == nil || gi.clock == nil {
		return time.Now()
	}
	return gi.clock.Now()
}

// SetClock replaces the clock returned by Now; nil restores the wall clock.
func (gi *GameInfo) SetClock(c Clock) {
	if c == nil {
		c = SystemClock{}
	}
	gi.clock = c
	if gi.State != nil {
		gi.State.clock = c
	}
}

func (gi GameInfo) PrintField() {
	fmt.Println(gi.field)
}

// ObserveRestart feeds this frame's ball position to the referee state so a
// pending kickoff, free kick or penalty is put in play (or times out) once for
// all teams. The runner calls it once per frame before any planner runs.
func (gi *GameInfo) ObserveRestart() {
	if gi.Status == nil || gi.State == nil {
		return
	}
	ball, ok := gi.State.GetTrackedBall().GetFreshTrackedPosition(gi.Now(), TrackedStaleAfter)
	if !ok {
		raw, err := gi.State.GetBall().GetPosition()
		ball, ok = raw, err == nil
	}
	gi.Status.GetGameEvent().ObserveRestart(ball, ok, gi.Now(), gi.Status.GetDivision())
}

func (gi *GameInfo) AdvanceVisionFrame() {
	gi.visionFrame++
}

func (gi GameInfo) VisionFrame() uint64 {
	return gi.visionFrame
}

// Existing lines:
// field_length:9000 field_width:6000 goal_width:1000 goal_depth:180 boundary_width:300
// field_lines:{name:"TopTouchLine" p1:{x:-4500 y:3000} p2:{x:4500 y:3000} thickness:10}
// field_lines:{name:"BottomTouchLine" p1:{x:-4500 y:-3000} p2:{x:4500 y:-3000} thickness:10}
// field_lines:{name:"LeftGoalLine" p1:{x:-4500 y:-3000} p2:{x:-4500 y:3000} thickness:10}
// field_lines:{name:"RightGoalLine" p1:{x:4500 y:-3000} p2:{x:4500 y:3000} thickness:10}
// field_lines:{name:"HalfwayLine" p1:{x:0 y:-3000} p2:{x:0 y:3000} thickness:10}
// field_lines:{name:"CenterLine" p1:{x:-4500 y:0} p2:{x:4500 y:0} thickness:10}
// field_lines:{name:"LeftPenaltyStretch" p1:{x:-3500 y:-1000} p2:{x:-3500 y:1000} thickness:10}
// field_lines:{name:"RightPenaltyStretch" p1:{x:3500 y:-1000} p2:{x:3500 y:1000} thickness:10}
// field_lines:{name:"LeftFieldLeftPenaltyStretch" p1:{x:-4500 y:-1000} p2:{x:-3500 y:-1000} thickness:10}
// field_lines:{name:"LeftFieldRightPenaltyStretch" p1:{x:-4500 y:1000} p2:{x:-3500 y:1000} thickness:10}
// field_lines:{name:"RightFieldRightPenaltyStretch" p1:{x:4500 y:-1000} p2:{x:3500 y:-1000} thickness:10}
// field_lines:{name:"RightFieldLeftPenaltyStretch" p1:{x:4500 y:1000} p2:{x:3500 y:1000} thickness:10}
// field_arcs:{name:"CenterCircle" center:{x:0 y:0} radius:500 a1:0 a2:6.2831855 thickness:10}

func (gi GameInfo) GetFieldLine(line string) *ssl_vision.SSL_FieldLineSegment {
	for i := range gi.field.FieldLines {
		if strings.EqualFold(gi.field.FieldLines[i].GetName(), line) {
			return gi.field.FieldLines[i]
		}
	}

	logger.Logger.Debugln(fmt.Sprintf("No fieldline %v found", line))

	return nil
}

/*
Return upper and lower point of enemy goal line as Position slice (x,y)
*/
func (gi GameInfo) EnemyGoalLine(team Team) []Position {
	x := -gi.OwnHalfXSign(team) * float64(gi.field.GetFieldLength()) / 2
	y := float64(gi.field.GetGoalWidth() / 2)

	upper := Position{X: x, Y: y, Z: 0, Angle: 0}
	lower := Position{X: x, Y: -y, Z: 0, Angle: 0}

	return []Position{upper, lower}
}

func (gi GameInfo) EnemyGoalCenter(team Team) Position {
	// Segfaults when not in sim
	opponentGoal := gi.EnemyGoalLine(team)[0].Add(&gi.EnemyGoalLine(team)[1])
	opponentGoal.Div2d(2.0)
	return opponentGoal
	// return Position{X: 3050, Y: 0, Z: 0, Angle: 0}
}

func (gi GameInfo) FieldSize() Position {
	x := float64(gi.field.GetFieldLength())
	y := float64(gi.field.GetFieldWidth())
	return Position{X: x, Y: y, Z: 0, Angle: 0}
}

func (gi GameInfo) FieldGeometry() (FieldGeometry, bool) {
	if !gi.HasField() {
		return FieldGeometry{}, false
	}

	geometry := FieldGeometry{
		Length:           float64(gi.field.GetFieldLength()),
		Width:            float64(gi.field.GetFieldWidth()),
		GoalWidth:        float64(gi.field.GetGoalWidth()),
		GoalDepth:        float64(gi.field.GetGoalDepth()),
		BoundaryWidth:    float64(gi.field.GetBoundaryWidth()),
		PenaltyAreaDepth: float64(gi.field.GetPenaltyAreaDepth()),
		PenaltyAreaWidth: float64(gi.field.GetPenaltyAreaWidth()),
	}
	return geometry, geometry.Length > 0 && geometry.Width > 0
}

func (gi GameInfo) FieldBoundaryWidth() float64 {
	if !gi.HasField() {
		return 0
	}
	return float64(gi.field.GetBoundaryWidth())
}

func (gi GameInfo) FieldBounds(margin float64) (minX, maxX, minY, maxY float64, ok bool) {
	return gi.FieldBoundsWithMargins(margin, margin)
}

// FieldBoundsWithMargins returns field bounds with independent goal-line (X)
// and touchline (Y) margins. Negative margins expand the corresponding bounds
// into the physical SSL boundary area.
func (gi GameInfo) FieldBoundsWithMargins(marginX, marginY float64) (minX, maxX, minY, maxY float64, ok bool) {
	if !gi.HasField() {
		return 0, 0, 0, 0, false
	}

	halfX := math.Max(0, float64(gi.field.GetFieldLength())/2-marginX)
	halfY := math.Max(0, float64(gi.field.GetFieldWidth())/2-marginY)
	return -halfX, halfX, -halfY, halfY, true
}

func (gi GameInfo) ClampToField(pos Position, margin float64) Position {
	return gi.ClampToFieldWithMargins(pos, margin, margin)
}

// ClampToFieldWithMargins clamps the goal-line and touchline axes
// independently. This lets normal field robots use the area beyond the
// touchlines without also permitting movement behind a goal line.
func (gi GameInfo) ClampToFieldWithMargins(pos Position, marginX, marginY float64) Position {
	minX, maxX, minY, maxY, ok := gi.FieldBoundsWithMargins(marginX, marginY)
	if !ok {
		return pos
	}

	pos.X = math.Max(minX, math.Min(maxX, pos.X))
	pos.Y = math.Max(minY, math.Min(maxY, pos.Y))
	return pos
}

// Division B dimensions, used until SSL-Vision has published geometry.
const (
	defaultFieldLength      = 9000.0
	defaultGoalWidth        = 1000.0
	defaultPenaltyAreaDepth = 1000.0
)

// PenaltyAreaDepth is the defense area's depth from the goal line, falling
// back to Division B until geometry arrives.
func (gi GameInfo) PenaltyAreaDepth() float64 {
	if geometry, ok := gi.FieldGeometry(); ok && geometry.PenaltyAreaDepth > 0 {
		return geometry.PenaltyAreaDepth
	}
	return defaultPenaltyAreaDepth
}

// FieldHalfLength is half the field length (goal line X), falling back to
// the Division B field when no geometry has arrived yet.
func (gi GameInfo) FieldHalfLength() float64 {
	if geometry, ok := gi.FieldGeometry(); ok {
		return geometry.Length / 2
	}
	return defaultFieldLength / 2
}

// OwnGoalCenter is the center of the goal team defends, on its goal line.
func (gi GameInfo) OwnGoalCenter(team Team) Position {
	return Position{X: gi.OwnHalfXSign(team) * gi.FieldHalfLength()}
}

// HomeGoalDefPos is the defensive anchor in front of team's goal, one goal
// width in from the goal line.
func (gi GameInfo) HomeGoalDefPos(team Team) Position {
	goalWidth := defaultGoalWidth
	if geometry, ok := gi.FieldGeometry(); ok && geometry.GoalWidth > 0 {
		goalWidth = geometry.GoalWidth
	}
	return Position{X: gi.OwnHalfXSign(team) * (gi.FieldHalfLength() - goalWidth)}
}

// OwnHalfXSign returns the X-axis sign of the half assigned to team by the
// referee. blue_team_on_positive_half indicates which goal belongs to blue;
// yellow is always assigned the opposite half.
func (gi GameInfo) OwnHalfXSign(team Team) float64 {
	blueOnPositiveHalf := false
	if gi.Status != nil {
		blueOnPositiveHalf = gi.Status.GetBlueTeamOnPositiveHalf()
	}

	switch team {
	case Blue:
		if blueOnPositiveHalf {
			return 1
		}
		return -1
	case Yellow:
		if blueOnPositiveHalf {
			return -1
		}
		return 1
	default:
		panic(fmt.Sprintf("Incorrect team given %v", team))
	}
}

func (gi GameInfo) HasField() bool {

	if gi.field == nil {
		return false
	}

	return true
}
func (gi *GameInfo) SetField(field *ssl_vision.SSL_GeometryFieldSize) {
	gi.field = field
}
