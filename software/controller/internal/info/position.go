package info

import (
	"fmt"
	"math"

	// "github.com/LiU-SeeGoals/controller/internal/info"
)

type Vec2 struct {
	X float64
	Y float64
}

type Position struct {
	X     float64
	Y     float64
	Z     float64
	Angle float64
}

func Sub(a Vec2, b Vec2) Vec2 {
	return Vec2{
		X: a.X - b.X,
		Y: a.Y - b.Y,
	}
}

func NormV2(a Vec2, b Vec2) Vec2 {
	return Vec2{
		X: a.X - b.X,
		Y: a.Y - b.Y,
	}
}

func (v *Vec2) String() string {
	return fmt.Sprintf("X: %f Y: %f", v.X, v.Y)
}

func (v *Vec2) Angle() float64 {
	return math.Atan2(v.Y, v.X)
}

func (v *Vec2) Mult(scalar float64) Vec2 {
	return Vec2{
		v.X * scalar,
		v.Y * scalar}
}

func Add(a Vec2, b Vec2) Vec2 {
	return Vec2{
		X: a.X + b.X,
		Y: a.Y + b.Y,
	}
}

func (p *Position) Div2d(scalar float64) {
	p.X = p.X / scalar
	p.Y = p.Y / scalar
}

func (v *Vec2) DivNorm() {
	norm := math.Sqrt(v.X*v.X + v.Y*v.Y)
	v.X = v.X / norm
	v.Y = v.Y / norm
}

func (v *Vec2) Norm() float64 {
	norm := math.Sqrt(v.X*v.X + v.Y*v.Y)
	return norm
}

func DistV2(a, b Vec2) float64 {
	xx := (a.X - b.X) * (a.X - b.X)
	yy := (a.Y - b.Y) * (a.Y - b.Y)

	return math.Sqrt(xx + yy)
}


func (p *Position) Dist2d(other Position) float64 {
	xx := (p.X - other.X) * (p.X - other.X)
	yy := (p.Y - other.Y) * (p.Y - other.Y)

	return math.Sqrt(xx + yy)
}

func Dist2d(a,b Position) float64 {
	xx := (a.X - b.X) * (a.X - b.X)
	yy := (a.Y - b.Y) * (a.Y - b.Y)

	return math.Sqrt(xx + yy)
}

func (p *Position) Norm2d() float64 {
	xx := (p.X) * (p.X)
	yy := (p.Y) * (p.Y)

	return math.Sqrt(xx + yy)
}

func (p *Position) FacingPosition(target Position, threshold float64) bool {

	targetDirection := p.AngleToPosition(target)
	currentDirection := p.Angle

	angleDiff := math.Abs(NormalizeAngleDelta(targetDirection, currentDirection))
	if angleDiff < threshold {
		return true
	} else {
		return false
	}
}

func NormalizeAngleDelta(target, current float64) float64 {
	delta := target - current
	return math.Atan2(math.Sin(delta), math.Cos(delta))
}

// Rotate position (vector) around origin (around z axis)
func (p Position) Rotate(rads float64) Position {

	x := p.X*math.Cos(rads) - p.Y*math.Sin(rads)
	y := p.X*math.Sin(rads) + p.Y*math.Cos(rads)
	angle := p.Angle + rads

	return Position{
		X:     x,
		Y:     y,
		Z:     0,
		Angle: angle,
	}
}

// TranslatePolar moves a point by a given distance in a given direction
func (p Position) OnRadius(distance float64, angle float64) Position {
	return Position{
		X:     p.X + distance*math.Cos(angle),
		Y:     p.Y + distance*math.Sin(angle),
		Z:     p.Z,
		Angle: p.Angle,
	}
}

func (p Position) AngleToPosition(p2 Position) float64 {
	dx := p2.X - p.X
	dy := p2.Y - p.Y
	return math.Atan2(dy, dx)
}

func (p Position) AngleDistance(p2 Position) float64 {
	diff := p.Angle - p2.Angle
	return math.Abs(math.Remainder(diff, 2*math.Pi))
}

// Disntance between two points
func (p Position) Distance(p2 Position) float64 {
	dx := p.X - p2.X
	dy := p.Y - p2.Y
	dz := p.Z - p2.Z
	return float64(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}

func (p Position) String() string {
	return fmt.Sprintf("(%f, %f, %f, %f)", p.X, p.Y, p.Z, p.Angle)
}

func (p Position) Add(other *Position) Position {
	return Position{
		X:     p.X + other.X,
		Y:     p.Y + other.Y,
		Z:     p.Z + other.Z,
		Angle: p.Angle + other.Angle,
	}
}

func (p Position) Sub(other *Position) Position {
	return Position{
		X:     p.X - other.X,
		Y:     p.Y - other.Y,
		Z:     p.Z - other.Z,
		Angle: p.Angle - other.Angle,
	}
}

func (p Position) Dot(other Position) float64 {
	return p.X*other.X + p.Y*other.Y + p.Z*other.Z
}

func DotV2(a, b Vec2) float64 {
	return a.X*b.X + a.Y*b.Y
}

func (p Position) Norm() float64 {
	return float64(math.Sqrt(float64(p.X*p.X + p.Y*p.Y + p.Z*p.Z)))
}

func (p Position) Scale(scalar float64) Position {
	return Position{
		X:     p.X * scalar,
		Y:     p.Y * scalar,
		Z:     p.Z * scalar,
		Angle: p.Angle * scalar,
	}
}

func (p Position) Normalize2d() Position {
	norm := p.Norm()
	return Position{
		X:     p.X/norm + 0.00001,
		Y:     p.Y/norm + 0.00001,
		Z:     p.Z,
		Angle: p.Angle,
	}
}

func (p Position) Normalize() Position {
	norm := p.Norm()
	return Position{
		X:     p.X / norm,
		Y:     p.Y / norm,
		Z:     p.Z / norm,
		Angle: p.Angle / norm,
	}
}

func (p Position) ToDTO() string {
	return fmt.Sprintf("(%f, %f, %f, %f)", p.X, p.Y, p.Z, p.Angle)
}

func (p Position) ToV2() Vec2 {
	return Vec2{p.X,p.Y}
}

func PointToLineSegment(a, b, c Vec2) Vec2 {

	ab := Sub(b, a)
	ac := Sub(c, a)

	t := DotV2(ac, ab) / DotV2(ab, ab)

	if t <= 0 {
		return a
	} else if t >= 1 {
		return b
	} else {
		return Add(a, ab.Mult(t))
	}
}

func DistToLineSegment(a, b, c Vec2) float64 {

	pointOnLine := PointToLineSegment(a,b,c)
	return DistV2(pointOnLine, c)
}
