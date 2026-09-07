package info

import "math"

// DefenseArea describes an uninflated rectangular area in vision coordinates.
// Callers apply their own robot/motion clearance to these common bounds.
type DefenseArea struct {
	FrontX, BackX, MinY, MaxY float64
}

func (gi GameInfo) DefenseAreas() []DefenseArea {
	if !gi.HasField() {
		return nil
	}
	areas := make([]DefenseArea, 0, 2)
	for _, names := range [][2]string{
		{"LeftPenaltyStretch", "LeftGoalLine"},
		{"RightPenaltyStretch", "RightGoalLine"},
	} {
		front, back := gi.GetFieldLine(names[0]), gi.GetFieldLine(names[1])
		if front == nil || back == nil || front.GetP1() == nil || front.GetP2() == nil || back.GetP1() == nil {
			continue
		}
		areas = append(areas, DefenseArea{
			FrontX: float64(front.GetP1().GetX()), BackX: float64(back.GetP1().GetX()),
			MinY: math.Min(float64(front.GetP1().GetY()), float64(front.GetP2().GetY())),
			MaxY: math.Max(float64(front.GetP1().GetY()), float64(front.GetP2().GetY())),
		})
	}
	geometry, ok := gi.FieldGeometry()
	if len(areas) == 2 || !ok || geometry.PenaltyAreaDepth <= 0 || geometry.PenaltyAreaWidth <= 0 {
		return areas
	}
	halfLength, halfWidth := geometry.Length/2, geometry.PenaltyAreaWidth/2
	return []DefenseArea{
		{FrontX: -halfLength + geometry.PenaltyAreaDepth, BackX: -halfLength, MinY: -halfWidth, MaxY: halfWidth},
		{FrontX: halfLength - geometry.PenaltyAreaDepth, BackX: halfLength, MinY: -halfWidth, MaxY: halfWidth},
	}
}
