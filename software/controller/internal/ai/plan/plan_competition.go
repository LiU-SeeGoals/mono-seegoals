package ai

import (
	"fmt"
	"sync"
	"time"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

// ========================================================
// plannerCompetition is a slow brain used for competition
// ========================================================

type plannerCompetition struct {
	plannerCore

	at_state int
	start    time.Time
	max_time time.Duration
	team     info.Team
	prev_ref info.RefCommand
}

func NewPlannerCompetition(team info.Team) *plannerCompetition {
	return &plannerCompetition{
		plannerCore: plannerCore{
			team: team,
		},
		team: team,
	}
}

func (m *plannerCompetition) Init(
	incoming <-chan info.GameInfo,
	activities *[info.TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	m.incomingGameInfo = incoming
	m.activities = activities // store pointer directly
	m.activity_lock = lock
	m.start = time.Now()

	go m.run()
}

// This is the main loop of the AI in this slow brain
func (m *plannerCompetition) run() {
	way_points := []info.Position{
		{X: 4600, Y: 0, Z: 0, Angle: 0},
		{X: -4600, Y: 0, Z: 0, Angle: 0},
	}

	enemy_goal := 0
	fmt.Println("plannerCompetition: starting")
	active_id := []int{0, 1, 3}

	for {
		// No need for slow brain to be fast
		time.Sleep(100 * time.Millisecond)

		gameInfo := <-m.incomingGameInfo

		if gameInfo.Status.GetBlueTeamOnPositiveHalf() && m.team == info.Blue || !gameInfo.Status.GetBlueTeamOnPositiveHalf() && m.team == info.Yellow {
			enemy_goal = 1
		} else {
			enemy_goal = 0

		}

		fmt.Println("blue on positive", gameInfo.Status.GetBlueTeamOnPositiveHalf())

		fmt.Println(gameInfo.Status.GetGameEvent().RefCommand)

		if m.HandleRef(&gameInfo, active_id) {
			continue
		}

		if m.activities[0] == nil {
			fmt.Println("done with action goalie: ", m.team)
			m.AddActivity(ai.NewGoalie(m.team, 0))
		}

		// The other robot is doing all the work

		// The logic for the other robot
		// 1. Chaise the ball
		// 2. If get the ball, dribble to a position in front of thier goal
		// 3. Kick the ball to the goal
		// 4. Repeat
		// fmt.Println(way_points[enemy_goal])

		if m.activities[1] == nil {
			fmt.Println("done with action: ", m.team)

			// If we have the ball, then dribble to the enemy goal
			possessor := gameInfo.State.GetBall().GetPossessor()

			if possessor != nil && possessor.GetID() == 1 {
				m.AddActivity(ai.NewMoveWithBallToPosition(m.team, 1, way_points[enemy_goal]))

			} else {
				m.AddActivity(ai.NewMoveToBall(m.team, 1))
			}
			// m.AddActivity(ai.NewRamAtPosition(m.team, 1, way_points[enemy_goal]))

		}

		if m.activities[3] == nil {
			fmt.Println("done with action: ", m.team)

			// // If we have the ball, then dribble to the enemy goal
			// possessor := gameInfo.State.GetBall().GetPossessor()
			//
			// if possessor != nil && possessor.GetID() == 2 {
			// 	m.AddActivity(ai.NewMoveWithBallToPosition(m.team, 2, way_points[enemy_goal]))
			//
			// } else {
			// 	m.AddActivity(ai.NewMoveToBall(m.team, 2))
			// }
			// m.AddActivity(ai.NewRamAtPosition(m.team, 2, way_points[enemy_goal]))

			// We are on positive half
			multiplier := float64(-1)
			if gameInfo.Status.GetBlueTeamOnPositiveHalf() && m.team == info.Blue || !gameInfo.Status.GetBlueTeamOnPositiveHalf() && m.team == info.Yellow {
				multiplier = float64(1)
			}

			activityLoop := []ai.Activity{
				ai.NewMoveToPosition(m.team, 3, info.Position{X: multiplier * 3000, Y: 1500, Z: 0, Angle: 0}),
				ai.NewMoveToPosition(m.team, 3, info.Position{X: multiplier * 3000, Y: -1500, Z: 0, Angle: 0}),
			}
			loop := ai.NewActivityLoop(3, activityLoop)
			m.AddActivity(loop)

		}
	}
}
