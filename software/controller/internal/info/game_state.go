package info

import (
	"encoding/json"
	"math"

	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

const TEAM_SIZE ID = 16

type GameState struct {
	Valid       bool
	Blue_team   *RobotTeam
	Yellow_team *RobotTeam

	Ball *Ball

	MessageReceived int64
}

// Constructor for game state
func NewGameState(capacity int) *GameState {
	gs := GameState{}
	gs.Valid = true

	gs.Ball = NewBall(capacity)
	var i ID
	gs.Blue_team = new(RobotTeam)
	gs.Yellow_team = new(RobotTeam)
	for i = 0; i < TEAM_SIZE; i++ {
		gs.Blue_team[i] = NewRobot(i, Blue, capacity)
		gs.Yellow_team[i] = NewRobot(i, Yellow, capacity)
	}
	return &gs
}

func (gs *GameState) ClosestRobot(target Position) (*Robot, float64) {
	shortestDistance := math.Inf(1)
	var closestRobot *Robot

	for _, robot := range gs.Blue_team {
		robotPos, err := robot.GetPosition()
		if err != nil {
			continue
		}

		distance := robotPos.Distance(target)
		if distance < shortestDistance {
			shortestDistance = distance
			closestRobot = robot
		}
	}
	for _, robot := range gs.Yellow_team {
		robotPos, err := robot.GetPosition()
		if err != nil {
			continue
		}

		distance := robotPos.Distance(target)
		if distance < shortestDistance {
			shortestDistance = distance
			closestRobot = robot
		}
	}

	return closestRobot, shortestDistance
}

// Should be called directly after inserting
// new data, recieved from ssl vision, into
// the game state
func (gs *GameState) Update() {

	// Update robot estimates
	// Has to be done before updating ball possessor
	for _, robot := range gs.Blue_team {
		if !robot.IsActive() {
			continue
		}
		// robot.Update()
	}
	for _, robot := range gs.Yellow_team {
		if !robot.IsActive() {
			continue
		}
		// robot.Update()
	}

	latestBallPos, _ := gs.Ball.GetPosition()
	latestPossessor := gs.Ball.GetPossessor()
	newPossessor := gs.FindBallPossessor()

	if gs.Ball.GetAge() < 50 { // Have a new ball measurement,  WARN: Magic number
		// fmt.Println("New ball measurement")
		gs.Ball.SetEstimatedPosition(latestBallPos)

		if latestPossessor != nil {
			if gs.LostBall(latestPossessor) { // We have a new measurement, so we can check if possessor has lost the ball
				gs.Ball.SetPossessor(nil)
			}
		}

		// If we have a new possessor (and either didn't have one before, or just lost it), update it.
		// Note: This allows instant "stealing" logic if we wanted, but primarily handles the case where latestPossessor was nil.
		if gs.Ball.GetPossessor() == nil && newPossessor != nil {
			gs.Ball.SetPossessor(newPossessor)
		}

	} else if latestPossessor != nil { // No new ball measurement, so we have to rely on the latest possessor

		// fmt.Println("No new ball measurement but we have a possessor")
		gs.Ball.SetEstimatedPosition(latestPossessor.DribblerPos())
	} else if newPossessor != nil { // A robot has arrived at the latest location of the ball
		// fmt.Println("No new ball measurement but we have a new possessor")
		gs.Ball.SetPossessor(newPossessor)
		gs.Ball.SetEstimatedPosition(newPossessor.DribblerPos())

	} else { // No new ball measurement and no possessor

		gs.Ball.SetEstimatedPosition(latestBallPos)
	}

}

func (gs *GameState) FindBallPossessor() *Robot {

	ballPos, err := gs.Ball.GetPosition()
	if err != nil {
		Logger.Errorf("Ball position retrieval failed: %v", err)
		return nil
	}

	// Get the robot closest to the ball
	closestToBall, _ := gs.ClosestRobot(ballPos)

	// Threshold for possession.
	const possessionThreshold = 30.0

	if closestToBall != nil {
		dist := ballPos.Distance(closestToBall.DribblerPos())
		// fmt.Printf("[DEBUG] Robot %d dist to ball dribbler: %.2f (threshold: %.2f)\n", closestToBall.ID(), dist, possessionThreshold)
		if dist < possessionThreshold {
			return closestToBall
		}
	}
	return nil
}

func (gs *GameState) LostBall(robot *Robot) bool {
	if robot == nil {
		Logger.Errorf("No robot")
		return true
	}

	ballPos, err := gs.Ball.GetPosition()
	if err != nil {
		Logger.Errorf("Ball position retrieval failed: %v", err)
		return true
	}

	farAway := ballPos.Distance(robot.DribblerPos()) > 50 // WARN: Magic number, mm

	// If the robot is not facing the ball
	// it has lost the ball
	return !robot.Facing(ballPos, 0.5) || farAway // WARN: Magic number

}

func (gs *GameState) SetValid(valid bool) {
	gs.Valid = valid
}

func (gs *GameState) IsValid() bool {
	return gs.Valid
}

func (gs *GameState) SetYellowRobot(robotId uint32, x, y, angle float64, time int64) {
	gs.Yellow_team[robotId].SetPositionTime(x, y, angle, time)
}

func (gs *GameState) SetBlueRobot(robotId uint32, x, y, angle float64, time int64) {
	gs.Blue_team[robotId].SetPositionTime(x, y, angle, time)
}

// Updates position of robots and balls to their actual position
func (gs *GameState) SetBall(x, y, z float64, time int64) {
	gs.Ball.SetPositionTime(x, y, z, time)
}

func (gs *GameState) SetMessageReceivedTime(time int64) {
	gs.MessageReceived = time

}

func (gs *GameState) GetMessageReceivedTime() int64 {
	return gs.MessageReceived
}

func (gs *GameState) GetBall() *Ball {
	return gs.Ball
}

func (gs *GameState) GetTeam(team Team) *RobotTeam {
	if team == Yellow {
		return gs.Yellow_team
	} else {
		return gs.Blue_team
	}
}

func (gs *GameState) GetOtherTeam(team Team) *RobotTeam {
	if team != Yellow {
		return gs.Yellow_team
	} else {
		return gs.Blue_team
	}
}

func (gs *GameState) GetYellowRobots() *RobotTeam {
	return gs.Yellow_team
}

func (gs *GameState) GetBlueRobots() *RobotTeam {
	return gs.Blue_team
}

func (gs *GameState) GetRobot(id ID, team Team) *Robot {
	return gs.GetTeam(team)[id]
}

// String representation of game state
func (gs *GameState) String() string {
	gs_str := "{\n blue team: {\n"
	var i ID
	for i = 0; i < TEAM_SIZE; i++ {
		gs_str += "robot: {" + gs.Blue_team[i].String() + " },\n"
	}
	gs_str += "},\n yellow team: {\n"
	for i = 0; i < TEAM_SIZE; i++ {
		gs_str += "robot: {" + gs.Yellow_team[i].String() + " },\n"
	}
	// for _, line := range gs.Field.FieldLines {
	// 	gs_str += fmt.Sprintf("line: {%s}\n", line.String())
	// }
	// for _, arc := range gs.Field.FieldArcs {
	// 	gs_str += fmt.Sprintf("arc: {%s}\n", arc.String())
	// }
	gs_str += "}"
	return gs_str
}

type GameStateDTO struct {
	RobotPositions [2 * TEAM_SIZE]RobotDTO
	BallPosition   BallDTO
}

func (gs GameState) ToDTO() *GameStateDTO {
	gameStateDTO := GameStateDTO{}
	var i ID
	for i = 0; i < TEAM_SIZE; i++ {
		gameStateDTO.RobotPositions[i] = gs.GetRobot(i, Blue).ToDTO()
	}
	for i = 0; i < TEAM_SIZE; i++ {
		gameStateDTO.RobotPositions[TEAM_SIZE+i] = gs.GetRobot(i, Yellow).ToDTO()
	}
	gameStateDTO.BallPosition = gs.Ball.ToDTO()
	return &gameStateDTO
}

func (gs GameState) ToJson() []byte {
	gameStateJson, err := json.Marshal(gs.ToDTO())
	if err != nil {
		// fmt.Println("The gamestate packet could not be marshalled to JSON.")
		Logger.Error("The gamestate packet could not be marshalled to JSON.")
	}
	return gameStateJson
}
