package info

import (
	"fmt"
	"math"
)

type Position struct {
	X     float64
	Y     float64
	Z     float64
	Angle float64
}

func (p *Position) FacingPosition(target Position, threshold float64) bool {

	targetDirection := p.AngleToPosition(target)
	currentDirection := p.Angle

	angleDiff := math.Abs(float64(targetDirection - currentDirection))
	if angleDiff < threshold {
		return true
	} else {
		return false
	}
}

// Rotate position (vector) around origin (around z axis)
func (p Position) Rotate(rads float64) Position {

	x := p.X * math.Cos(rads) - p.Y * math.Sin(rads)
	y := p.X * math.Sin(rads) + p.Y * math.Cos(rads)
	angle := p.Angle + rads

	return Position{
		X: x,
		Y: y,
		Z: 0,
		Angle: angle,
	}
}

// TranslatePolar moves a point by a given distance in a given direction
func (p Position) OnRadius(distance float64, angle float64) Position {
	return Position{
		X: p.X + distance*math.Cos(angle),
		Y: p.Y + distance*math.Sin(angle),
		Z: p.Z,
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
