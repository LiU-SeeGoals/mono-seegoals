package ai

import (
	"math/rand"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/simulator"
)

type TestKick struct {
	plannerCore
	at_state   int
	start      time.Time
	max_time   time.Duration
	simControl *simulator.SimControl
}

func NewTestKick(team info.Team, simControl *simulator.SimControl) *TestKick {
	return &TestKick{
		plannerCore: plannerCore{
			team: team,
		},
		simControl: simControl,
	}
}

func (m *TestKick) Init(
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

func wait(ms time.Duration) {
	time.Sleep(ms)
}

func (g *TestKick) run() {
	// Field boundaries ?? TODO: check if this is correct (in mm)
	const fieldHalfLength = 4500.0
	const fieldHalfWidth = 3000.0
	for {
		wait(100 * time.Millisecond)

		gameInfo := <-g.incomingGameInfo
		ballPos, err := gameInfo.State.GetBall().GetPosition()
		if err == nil {
			// Check if ball is outside field boundaries
			if ballPos.X < -fieldHalfLength || ballPos.X > fieldHalfLength ||
				ballPos.Y < -fieldHalfWidth || ballPos.Y > fieldHalfWidth {
				// Teleport ball to random position inside field
				x := float32(rand.Intn(6000) - 3000) // -3000 to 3000
				y := float32(rand.Intn(4000) - 2000) // -2000 to 2000
				g.simControl.TeleportBall(x, y)
			}
		}

		if g.activities[0] == nil {
			queue := ai.NewActivityQueue(0, []ai.Activity{
				ai.NewKickAtPosition(g.team, 0, info.Position{X: 2000, Y: 500}),
			})
			g.AddActivity(queue)
		}
	}

}
