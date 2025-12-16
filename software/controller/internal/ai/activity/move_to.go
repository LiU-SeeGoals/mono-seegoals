
package ai

import (
	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type MoveTo struct {
	team              info.Team
	id                info.ID
	dest 					info.Position
}

func NewMoveTo(team info.Team, id info.ID, dest info.Position) *MoveTo{
	// Initialize with reasonable RRT parameters
	return &MoveTo{
		team,
		id,
		dest,
	}
}

// GetAction returns an action for the robot with RRT-based collision avoidance
func (m *MoveTo) GetAction(gi *info.GameInfo) action.Action {
	moveToAction := m.GetMoveTo(gi)
	return &moveToAction
}

func (m *MoveTo) GetMoveTo(gi *info.GameInfo) action.MoveTo {

	robotId := 3
	myRobot := gi.State.GetTeam(m.team)[robotId]
	myPos, _ := myRobot.GetPosition()

	act := action.MoveTo{}
	act.Id = int(m.id)
	act.Team = m.team
	act.Pos = myPos
	act.Dest = m.dest
	act.Dest.Angle = m.dest.Angle
	act.Dribble = false

	return act
}

// Achieved returns true if the robot is sufficiently close to the final destination
func (m *MoveTo) Achieved(gi *info.GameInfo) bool {
	// currPos, _ := gi.State.GetTeam(m.team)[m.id].GetPosition()
	// distanceLeft := distanceBetween(currPos, m.final_destination)
	// completionRadius := 50
	return false
}

func (m *MoveTo) String() string {
	return "yeet"
}

func (m *MoveTo)GetID() info.ID{
	return m.id
}
