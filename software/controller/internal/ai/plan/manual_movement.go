package plan

import (
	"sync"

	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type plannerManualMovement struct {
	plannerCore
}

func NewPlannerManualMovement(team info.Team) *plannerManualMovement {
	return &plannerManualMovement{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *plannerManualMovement) Init(
	activities *[info.TEAM_SIZE]activity.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	m.ActivityHandler.Activities = activities // store pointer directly
	m.ActivityHandler.Activity_lock = lock
	m.team = team
	m.Active = true
}

// Tick runs before the executor receives this control frame, so a click can
// change its activity in the same frame.
func (m *plannerManualMovement) Tick(_ *info.GameInfo) {
	if !m.Active {
		return
	}
	command := client.TakeCommandFor(client.MOVE_ROBOT, int(m.team))
	if command == nil || command.Id < 0 || command.Id >= int(info.TEAM_SIZE) {
		return
	}
	pos := info.Position{X: float64(command.X), Y: float64(command.Y)}
	m.ActivityHandler.AddActivity(activity.NewMoveToPosition(m.team, info.ID(command.Id), pos))
}

func (m *plannerManualMovement) Kill() {
	m.Active = false
}
