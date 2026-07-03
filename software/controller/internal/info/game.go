package info

import (
	"fmt"
	"math"
	"strings"

	"github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
)

// Maybe State should also be private, so we can keep track of coordinate system here?
type GameInfo struct {
	State       *GameState
	Status      *GameStatus
	field       *ssl_vision.SSL_GeometryFieldSize
	visionFrame uint64
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
	}
}

func (gi GameInfo) PrintField() {
	fmt.Println(gi.field)
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

func (gi GameInfo) HomeGoalDefPos(team Team) Position {
	x := gi.OwnHalfXSign(team) * float64(gi.field.GetFieldLength()/2-gi.field.GetGoalWidth())

	pos := Position{X: x, Y: 0, Z: 0, Angle: 0}

	return pos
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
