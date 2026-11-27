package info

import (
	"fmt"
	"math"
)

// Vec2 is a plain 2D vector (a point, offset or velocity) with no heading.
// Prefer it for geometry; Position mixes a point with a heading.
type Vec2 struct {
	X float64
	Y float64
}

// Pose2D is a robot pose: a field position plus a heading in radians.
type Pose2D struct {
	Pos     Vec2
	Heading float64
}

func (v Vec2) Plus(o Vec2) Vec2             { return Vec2{v.X + o.X, v.Y + o.Y} }
func (v Vec2) Minus(o Vec2) Vec2            { return Vec2{v.X - o.X, v.Y - o.Y} }
func (v Vec2) Scaled(k float64) Vec2        { return Vec2{v.X * k, v.Y * k} }
func (v Vec2) Dot(o Vec2) float64           { return v.X*o.X + v.Y*o.Y }
func (v Vec2) Cross(o Vec2) float64         { return v.X*o.Y - v.Y*o.X }
func (v Vec2) Len() float64                 { return math.Hypot(v.X, v.Y) }
func (v Vec2) DistTo(o Vec2) float64        { return v.Minus(o).Len() }
func (v Vec2) Heading() float64             { return math.Atan2(v.Y, v.X) }
func (v Vec2) WithHeading(h float64) Pose2D { return Pose2D{Pos: v, Heading: h} }

// Unit returns v scaled to length 1; the zero vector stays zero.
func (v Vec2) Unit() Vec2 {
	n := v.Len()
	if n == 0 {
		return Vec2{}
	}
	return Vec2{v.X / n, v.Y / n}
}

// Rotated rotates v counter-clockwise by rads around the origin.
func (v Vec2) Rotated(rads float64) Vec2 {
	c, s := math.Cos(rads), math.Sin(rads)
	return Vec2{v.X*c - v.Y*s, v.X*s + v.Y*c}
}

// Position converts the pose to the legacy Position type (Z = 0).
func (p Pose2D) Position() Position {
	return Position{X: p.Pos.X, Y: p.Pos.Y, Angle: p.Heading}
}

// HeadingError is the signed shortest rotation from p's heading to target.
func (p Pose2D) HeadingError(target float64) float64 {
	return NormalizeAngleDelta(target, p.Heading)
}

// Position is the legacy combined point/pose type used across the AI: X, Y,
// Z in mm plus Angle (heading) in radians. Use XY() or Pose() to get the
// typed views for new code.
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

func NormV2(v Vec2) float64 {
	norm := math.Sqrt(v.X*v.X + v.Y*v.Y)
	return norm
}

func (p *Position) Dist2d(other Position) float64 {
	xx := (p.X - other.X) * (p.X - other.X)
	yy := (p.Y - other.Y) * (p.Y - other.Y)

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

// Deprecated: Add also applies to Angle componentwise, which is wrong for
// headings (no wrap-around). Use XY() and the Vec2 methods for vectors.
func (p Position) Add(other *Position) Position {
	return Position{
		X:     p.X + other.X,
		Y:     p.Y + other.Y,
		Z:     p.Z + other.Z,
		Angle: p.Angle + other.Angle,
	}
}

// Deprecated: Sub also applies to Angle componentwise, which is wrong for
// headings (no wrap-around). Use XY() and the Vec2 methods for vectors.
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

// Deprecated: Scale also applies to Angle componentwise, which is wrong for
// headings (no wrap-around). Use XY() and the Vec2 methods for vectors.
func (p Position) Scale(scalar float64) Position {
	return Position{
		X:     p.X * scalar,
		Y:     p.Y * scalar,
		Z:     p.Z * scalar,
		Angle: p.Angle * scalar,
	}
}

// Normalize2d returns the unit vector of (X, Y), keeping Z and Angle.
// A zero vector stays zero instead of becoming NaN.
func (p Position) Normalize2d() Position {
	norm := math.Hypot(p.X, p.Y)
	if norm == 0 {
		return Position{Z: p.Z, Angle: p.Angle}
	}
	return Position{
		X:     p.X / norm,
		Y:     p.Y / norm,
		Z:     p.Z,
		Angle: p.Angle,
	}
}

// Normalize returns the unit vector of (X, Y, Z), keeping Angle.
// A zero vector stays zero instead of becoming NaN.
func (p Position) Normalize() Position {
	norm := p.Norm()
	if norm == 0 {
		return Position{Angle: p.Angle}
	}
	return Position{
		X:     p.X / norm,
		Y:     p.Y / norm,
		Z:     p.Z / norm,
		Angle: p.Angle,
	}
}

func (p Position) ToDTO() string {
	return fmt.Sprintf("(%f, %f, %f, %f)", p.X, p.Y, p.Z, p.Angle)
}

func (p Position) ToV2() Vec2 {
	return Vec2{p.X, p.Y}
}

// XY is the position without Z or heading.
func (p Position) XY() Vec2 { return Vec2{p.X, p.Y} }

// Pose is the 2D pose (X, Y, Angle).
func (p Position) Pose() Pose2D { return Pose2D{Pos: Vec2{p.X, p.Y}, Heading: p.Angle} }

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

	pointOnLine := PointToLineSegment(a, b, c)
	return DistV2(pointOnLine, c)
}
