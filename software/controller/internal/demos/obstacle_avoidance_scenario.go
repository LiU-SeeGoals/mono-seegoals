// controller/internal/demos/obstacle_avoidance_scenario.go
package demos

import (
	"fmt"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	plan "github.com/LiU-SeeGoals/controller/internal/ai/plan"
	"github.com/LiU-SeeGoals/controller/internal/client"
	"github.com/LiU-SeeGoals/controller/internal/config"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/simulator"
)

func ObstacleAvoidanceScenario() {
	fmt.Println("Starting Obstacle Avoidance Scenario...")

	// --- 1. Setup the Simulator Scene ---
	simController := simulator.NewSimControl()

	// Define which robots are on the field.
	// Yellow 1 is our moving robot. Blue 0, 2, 3, 4 are static obstacles.
	presentYellow := []int{0,1,2,3,4,5,6,7,8,9,10}
	presentBlue := []int{2}
	simController.SetPresentRobots(presentYellow, presentBlue)
	
	// Give the simulator a moment to place the robots
	time.Sleep(100 * time.Millisecond)

	// Place our test robot at its starting position.
	simController.TeleportRobot(-4000, 0, 0, info.Yellow)
	simController.TeleportRobot(4000, 0, 1, info.Yellow)
	simController.TeleportRobot(0, 2000, 2, info.Yellow)
	simController.TeleportRobot(0, -2000, 3, info.Yellow)
	simController.TeleportRobot(0, -1000, 4, info.Yellow)
	simController.TeleportRobot(0, 1000, 5, info.Yellow)
	simController.TeleportRobot(2000, 0, 6, info.Yellow)
	simController.TeleportRobot(-2000, 0, 7, info.Yellow)
	simController.TeleportRobot(0, 3000, 8, info.Yellow)
	simController.TeleportRobot(0, -3000, 9, info.Yellow)
	simController.TeleportRobot(3000, 0, 10, info.Yellow)

	// Place the obstacle robots in a line to block the path.
	// simController.TeleportRobot(0, 1000, 1, info.Blue)
	simController.TeleportRobot(0, 0, 2, info.Blue)
	// simController.TeleportRobot(0, -1000, 3, info.Blue)
	// simController.TeleportRobot(0, 11500, 3, info.Blue)
	// simController.TeleportRobot(0, 11500, 4, info.Blue)

	// --- 2. Initialize AI and Clients ---
	gameInfo := info.NewGameInfo(10)
	ssl_receiver := client.NewSSLClient(config.GetSSLClientAddress())

	// Use our new Obstacle Avoidance planner for the Yellow team.
	slowBrainYellow := plan.NewPlannerObstacleAvoidance(info.Yellow)
	fastBrainYellow := ai.NewActivityExecutor()
	aiYellow := ai.NewAi(info.Yellow, slowBrainYellow, fastBrainYellow)
	// same for blue if needed

	// Send commands to the simulator.
	simClientYellow := client.NewSimClient(config.GetSimYellowTeamAddress(), gameInfo)

	// --- 3. Run the Main Loop ---
	startTime := time.Now().UnixMilli()
	for {
		playTime := time.Now().UnixMilli() - startTime

		// Get the latest game state from the vision client.
		ssl_receiver.UpdateState(gameInfo, playTime)
		// yellowRobot := gameInfo.State.GetRobot(1, info.Yellow)
		// blueObstacle := gameInfo.State.GetRobot(5, info.Blue)
	
		// if yellowRobot.IsActive() && blueObstacle.IsActive() {
		// 	yPos, _ := yellowRobot.GetPosition()
		// 	bPos, _ := blueObstacle.GetPosition()
		// 	fmt.Printf("AI sees Yellow 1 at (%.0f, %.0f) and Blue 0 at (%.0f, %.0f)\n", yPos.X, yPos.Y, bPos.X, bPos.Y)
		// }
		// Get actions from our AI.
		yellow_actions := aiYellow.GetActions(gameInfo)

		// Send the actions to the simulator.
		simClientYellow.SendActions(yellow_actions)

		// Optional: Broadcast state to the Game Viewer
		client.UpdateWebGUI(gameInfo.State, yellow_actions, []string{"Obstacle Avoidance Test"})
	}
}