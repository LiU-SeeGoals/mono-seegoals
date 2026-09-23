package ai

import (
	"sync"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
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
	incoming <-chan info.GameInfo,
	activities *[info.TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	m.incomingGameInfo = incoming
	m.ActivityHandler.Activities = activities // store pointer directly
	m.ActivityHandler.Activity_lock = lock
	m.team = team
	m.Active = true
}

// ApplyPendingCommand is called before the executor receives this control
// frame, so a click can change its activity in the same frame.
func (m *plannerManualMovement) ApplyPendingCommand() {
	if !m.Active {
		return
	}
	command := client.GetCommand(client.MOVE_ROBOT)
	if command == nil || command.Id < 0 || command.Id >= int(info.TEAM_SIZE) {
		return
	}
	pos := info.Position{X: float64(command.X), Y: float64(command.Y)}
	m.ActivityHandler.AddActivity(ai.NewMoveToPosition(m.team, info.ID(command.Id), pos))
}

func (m *plannerManualMovement) Kill() {
	m.Active = false
}
