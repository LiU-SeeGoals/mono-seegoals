package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type plannerDefence struct {
	plannerCore
	at_state int
	start    time.Time
	max_time time.Duration
}

// NewPlannerDefence returns a planner that mirrors planner1 but adds a defender on robot #4.
func NewPlannerDefence(team info.Team) *plannerDefence {
	return &plannerDefence{
		plannerCore: plannerCore{
			team: team,
		},
	}
}

// opponentCloseToBall reports whether an opposing robot is within threshold of the ball.
func opponentCloseToBall(gi info.GameInfo, myTeam info.Team, threshold float64) bool {
	if gi.State == nil {
		return false
	}
	// Get ball
	ballPos, err := gi.State.GetBall().GetEstimatedPosition()
	if err != nil {
		return false
	}
	// Get opponent
	opponentTeam := myTeam.Opponent()
	for _, r := range gi.State.GetTeam(opponentTeam) {
		if r == nil {
			continue
		}
		pos, err := r.GetPosition()
		if err != nil {
			continue
		}

		// Execute
		if pos.Distance(ballPos) < threshold {
			return true
		}
	}
	return false
}

func (m *plannerDefence) Init(
	incoming <-chan info.GameInfo,
	activities *[info.TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	m.incomingGameInfo = incoming
	m.activities = activities
	m.activity_lock = lock
	m.team = team
	m.start = time.Now()

	go m.run()
}

// run is the main loop of this planner; mirrors planner1 timing/behaviour.
func (m *plannerDefence) run() {

	var latestGI info.GameInfo

	for {
		// No need for slow brain to be fast
		time.Sleep(100 * time.Millisecond)

		// Get the newest game info
		select {
		case gi := <-m.incomingGameInfo:
			latestGI = gi
		default:
		}
		if latestGI.State == nil {
			continue
		}

		// Defender logic for id 4 on both teams
		if opponentCloseToBall(latestGI, m.team, 300) {
			if m.activities[4] == nil {
				ballPos, err := latestGI.State.GetBall().GetEstimatedPosition()
				if err != nil {
					ballPos = info.Position{}
				}

				// Get the goal position
				goalX := -4300.0
				if m.team == info.Yellow {
					goalX = 4300.0
				}

				// Get the target position
				target := info.Position{
					X:     (goalX + ballPos.X) / 2,
					Y:     ballPos.Y / 2,
					Z:     0,
					Angle: 0,
				}

				fmt.Printf("ball=(%.1f, %.1f), defender target=(%.1f, %.1f)\n", ballPos.X, ballPos.Y, target.X, target.Y)
				m.AddActivity(ai.NewMoveToPosition(m.team, 4, target))
			}
		} else {
			m.activities[4] = nil
		}

		// Blues activities are done separately from yellows; skip yellow logic.
		if m.team == info.Blue {
			continue
		}

		if m.activities[2] == nil {
			fmt.Println("done with action: ", m.team)
			m.AddActivity(ai.NewMoveToBall(m.team, 2))
		}
	}
}
