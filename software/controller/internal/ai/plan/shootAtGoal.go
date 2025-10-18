package ai

import (
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type ShootAtGoal struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

func NewShootAtGoal(team info.Team) *ShootAtGoal {
	return &ShootAtGoal{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

func (m *ShootAtGoal) Init(
	incoming <-chan info.GameInfo,
	activities *[info.TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	m.incomingGameInfo = incoming
	m.activities = activities // store pointer directly
	m.activity_lock = lock
	m.team = team
	m.start = time.Now()

	go m.run()
}

// This is the main loop of the AI in this slow brain
func (m *ShootAtGoal) run() {
	for {
		time.Sleep(100 * time.Millisecond)

		if m.activities[2] == nil {

			activityLoop := []ai.Activity{
				ai.NewMoveToBall(m.team, 2),
				ai.NewKickAtPosition(m.team, 2, info.Position{X: 2000, Y: 2000, Z: 0, Angle: 0}),
				// ai.NewKickToPlayer(m.team, 2, 1),
			}
			loop := ai.NewActivityLoop(2, activityLoop)
			m.AddActivity(loop)
		}

	}

}
