package ai

import (
	"sync"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/simulator"
)

type RefCommands struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
	simControl *simulator.SimControl
}

func NewRefCommands(team info.Team, simControl *simulator.SimControl) *RefCommands {
	return &RefCommands{
		plannerCore: plannerCore{
			team: team,
		},
		simControl: simControl,

	}
}

func (m *RefCommands) Init(
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
func (m *RefCommands) run() {
	robotId := info.ID(0)
	// gi := <-m.incomingGameInfo

	for {
		// fmt.Println(gi.Status.GetGameEvent().GetRefCommand())
		// fmt.Println(gi.Status.GetGameEvent().GetNextCommand())
		time.Sleep(100 * time.Millisecond)

		if m.activities[robotId] == nil {
			// robotPos, _ := gi.State.GetRobot(robotId, m.team).GetPosition()

			// m.simControl.TeleportBall(float32(robotPos.X+300), float32(robotPos.Y))
			

			activityLoop := []ai.Activity{
				ai.NewRefKickoff(robotId, m.team),
			}
			loop := ai.NewActivityLoop(robotId, activityLoop)
			m.AddActivity(loop)
		}

	}

}
