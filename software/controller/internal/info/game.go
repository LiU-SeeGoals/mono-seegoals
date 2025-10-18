package info

import (
	"fmt"
	"math"
	"github.com/LiU-SeeGoals/controller/internal/logger"
	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
)

// Maybe State should also be private, so we can keep track of coordinate system here?
type GameInfo struct {
	State  *GameState
	Status *GameStatus
	field  *ssl_vision.SSL_GeometryFieldSize
}

func NewGameInfo(capacity int) *GameInfo {
	return &GameInfo{
		State:  NewGameState(capacity),
		Status: NewGameStatus(),
		field:  nil,
	}
}

// Rotates all positions so that no matter 
// which side we are on we can use the same coordinate system
func correctedPosition(team Team, pos Position) Position{

	if team == Yellow{
		return pos
	} else if team == Blue{
		return pos.Rotate(math.Pi)
	} else {
		panic(fmt.Sprintf("Incorrect team given %v", team))
	}
}

func (gi GameInfo) PrintField() {
	fmt.Println(gi.field)
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

	for i := range gi.field.FieldLines{
		if *gi.field.FieldLines[i].Name == "CenterLine"{
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


	// x := float64(gi.field.GetFieldLength()/2 - gi.field.GetGoalWidth())

	center_line := gi.GetFieldLine("centerline").GetP1()

	x := float64(*center_line.X)
	y := float64(gi.field.GetGoalWidth()/2)

	upper := Position{X: x, Y: y, Z: 0, Angle: 0}
	lower := Position{X: x, Y: -y, Z: 0, Angle: 0}


	// upper := mat.NewVecDense(2, []float64{x, y})
	// lower := mat.NewVecDense(2, []float64{x, -y})

	return []Position{correctedPosition(team, upper), correctedPosition(team, lower)}
}
func (gi GameInfo) FieldSize() Position {
	x := float64(gi.field.GetFieldLength())
	y := float64(gi.field.GetFieldWidth())
	return Position{X: x, Y: y, Z: 0, Angle: 0}
}

func (gi GameInfo) HomeGoalDefPos(team Team) Position {

	x := float64(gi.field.GetFieldLength()/2 - gi.field.GetGoalWidth())

	pos := Position{X: x, Y: 0, Z: 0, Angle: 0}

	return correctedPosition(team, pos)
}

func (gi GameInfo) HasField() bool {

	if gi.field == nil{
		return false
	}

	return true
}
func (gi *GameInfo) SetField(field *ssl_vision.SSL_GeometryFieldSize) {
	gi.field = field
}
