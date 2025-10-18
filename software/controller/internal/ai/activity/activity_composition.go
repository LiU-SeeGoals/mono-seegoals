package ai

import (

	"github.com/LiU-SeeGoals/controller/internal/info"
)

type GenericComposition struct {
	team info.Team
	id   info.ID
}

// Here we have funciton that are common across multiple activities,
// such as calculating the distance between two points.
// or movement that is legal and not blocked by other players.


// func () CalculateDistance(p1, p2 info.Position) float32 {
// 	dx := p1.X - p2.X
// 	dy := p1.Y - p2.Y
// 	dz := p1.Z - p2.Z
// 	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
// }
