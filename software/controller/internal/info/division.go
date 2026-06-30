package info

import "fmt"

type Division int

const (
	DivisionUnknown Division = iota
	DivisionA
	DivisionB
)

func (d Division) String() string {
	switch d {
	case DivisionA:
		return "Division A"
	case DivisionB:
		return "Division B"
	case DivisionUnknown:
		return "Unknown Division"
	default:
		return fmt.Sprintf("Unknown Division (%d)", d)
	}
}
