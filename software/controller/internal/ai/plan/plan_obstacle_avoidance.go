// controller/internal/ai/plan/plan_obstacle_avoidance.go
package ai

import (
	"fmt"
	"sync"
	"time"
	"math/rand"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

// plannerObstacleAvoidance is a simple plan for testing collision avoidance.
type plannerObstacleAvoidance struct {
	plannerCore
	waypoints []info.Position
	waypointIndex int
}

func NewPlannerObstacleAvoidance(team info.Team) *plannerObstacleAvoidance {
	// Define two points on opposite sides of the field for the robot to travel between.
	waypoints := []info.Position{
		{X: 4000, Y: 0, Z: 0, Angle: 0},
		{X: -4000, Y: 0, Z: 0, Angle: 0},
	}

	return &plannerObstacleAvoidance{
		plannerCore: plannerCore{
			team: team,
		},
		waypoints: waypoints,
		waypointIndex: 0,
	}
}

func (p *plannerObstacleAvoidance) Init(
	incoming <-chan info.GameInfo,
	activities *[info.TEAM_SIZE]ai.Activity,
	lock *sync.Mutex,
	team info.Team,
) {
	p.incomingGameInfo = incoming
	p.activities = activities
	p.activity_lock = lock
	p.team = team
	go p.run()
}

// The main loop for this planner.
func (p *plannerObstacleAvoidance) run() {

	//robotID := info.ID(rand.Intn(2)) // We will test with Robot 1

	fmt.Println("Obstacle Avoidance Planner: Started")

	for {
		time.Sleep(100 * time.Millisecond)
		for robotID := info.ID(0); robotID < info.TEAM_SIZE; robotID++ {

		// Check if the current activity for our robot is finished (is nil).
		if p.activities[robotID] == nil {
			// The previous move is complete, so let's assign a new one.
			destination := info.Position{
				X:     float64(rand.Intn(8000) - 4000), // Random X between -4000 and 4000
				Y:     float64(rand.Intn(6000) - 3000), // Random Y between -3000 and 3000
				Z:     0,
				Angle: 0,
			}
			fmt.Printf("Obstacle Avoidance Planner: Robot %d finished. New target: (%.0f, %.0f)\n", robotID, destination.X, destination.Y)

			// Create and add the MoveToPosition activity.
			// This activity has the built-in RRT collision avoidance logic.
			moveActivity := ai.NewMoveToPosition(p.team, robotID, destination)
			p.AddActivity(moveActivity)

			// Move to the next waypoint for the next cycle.
			p.waypointIndex = (p.waypointIndex + 1) % len(p.waypoints)
		}
	}
	}
}