package info

import (
	"encoding/json"
	"math"

	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

const TEAM_SIZE ID = 16

type KickedBallInfo struct {
	StartX    float64
	StartY    float64
	StartTS   float64
	StopX     float64
	StopY     float64
	StopTS    float64
	RobotTeam Team
	RobotId   uint32
	Valid     bool
}

type GameState struct {
	Valid       bool
	Blue_team   *RobotTeam
	Yellow_team *RobotTeam

	Ball *Ball

	TrackedBlue   [TEAM_SIZE]*TrackedRobot
	TrackedYellow [TEAM_SIZE]*TrackedRobot
	TrackedBall   *TrackedBall

	KickedBall KickedBallInfo

	MessageReceived int64
	Timestamp       float64
}

func NewGameState(capacity int) *GameState {
	gs := GameState{}
	gs.Valid = true

	gs.Ball = NewBall(capacity)
	gs.TrackedBall = NewTrackedBall()

	var i ID
	gs.Blue_team = new(RobotTeam)
	gs.Yellow_team = new(RobotTeam)
	for i = 0; i < TEAM_SIZE; i++ {
		gs.Blue_team[i] = NewRobot(i, Blue, capacity)
		gs.Yellow_team[i] = NewRobot(i, Yellow, capacity)

		gs.TrackedBlue[i] = NewTrackedRobot(uint32(i), Blue)
		gs.TrackedYellow[i] = NewTrackedRobot(uint32(i), Yellow)
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

func (gs *GameState) Update() {
	for _, robot := range gs.Blue_team {
		if !robot.IsActive() {
			continue
		}
	}
	for _, robot := range gs.Yellow_team {
		if !robot.IsActive() {
			continue
		}
	}

	latestBallPos, _ := gs.Ball.GetPosition()
	latestPossessor := gs.Ball.GetPossessor()
	newPossessor := gs.FindBallPossessor()

	if gs.Ball.GetAge() < 50 {
		gs.Ball.SetEstimatedPosition(latestBallPos)

		if latestPossessor != nil {
			if gs.LostBall(latestPossessor) {
				gs.Ball.SetPossessor(nil)
			}
		}

		if gs.Ball.GetPossessor() == nil && newPossessor != nil {
			gs.Ball.SetPossessor(newPossessor)
		}

	} else if latestPossessor != nil {
		gs.Ball.SetEstimatedPosition(latestPossessor.DribblerPos())
	} else if newPossessor != nil {
		gs.Ball.SetPossessor(newPossessor)
		gs.Ball.SetEstimatedPosition(newPossessor.DribblerPos())

	} else {
		gs.Ball.SetEstimatedPosition(latestBallPos)
	}
}

func (gs *GameState) FindBallPossessor() *Robot {
	ballPos, err := gs.Ball.GetPosition()
	if err != nil {
		Logger.Errorf("Ball position retrieval failed: %v", err)
		return nil
	}

	closestToBall, _ := gs.ClosestRobot(ballPos)

	const possessionThreshold = 30.0

	if closestToBall != nil {
		dist := ballPos.Distance(closestToBall.DribblerPos())
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

	farAway := ballPos.Distance(robot.DribblerPos()) > 50

	return !robot.Facing(ballPos, 0.5) || farAway
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

func (gs *GameState) SetBall(x, y, z float64, time int64) {
	gs.Ball.SetPositionTime(x, y, z, time)
}

func (gs *GameState) SetMessageReceivedTime(time int64) {
	gs.MessageReceived = time
}

func (gs *GameState) GetMessageReceivedTime() int64 {
	return gs.MessageReceived
}

func (gs *GameState) SetTimestamp(ts float64) {
	gs.Timestamp = ts
}

func (gs *GameState) SetTrackedRobot(team Team, id uint32, pos Position, vel Position, angle float64, velAngular float64, ts float64) {
	switch team {
	case Yellow:
		gs.TrackedYellow[id].SetTracked(pos, vel, angle, velAngular, ts)
	case Blue:
		gs.TrackedBlue[id].SetTracked(pos, vel, angle, velAngular, ts)
	}
}

func (gs *GameState) SetTrackedBall(pos Position, vel Position, ts float64) {
	gs.TrackedBall.SetTracked(pos, vel, ts)
}

func (gs *GameState) GetTrackedRobot(team Team, id uint32) *TrackedRobot {
	switch team {
	case Yellow:
		return gs.TrackedYellow[id]
	case Blue:
		return gs.TrackedBlue[id]
	default:
		return nil
	}
}

func (gs *GameState) GetTrackedBall() *TrackedBall {
	return gs.TrackedBall
}

func (gs *GameState) SetKickedBall(startX, startY, startTS, stopX, stopY, stopTS float64, team Team, id uint32) {
	gs.KickedBall = KickedBallInfo{
		StartX:    startX,
		StartY:    startY,
		StartTS:   startTS,
		StopX:     stopX,
		StopY:     stopY,
		StopTS:    stopTS,
		RobotTeam: team,
		RobotId:   id,
		Valid:     true,
	}
}

func (gs *GameState) ClearKickedBall() {
	gs.KickedBall.Valid = false
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
		Logger.Error("The gamestate packet could not be marshalled to JSON.")
	}
	return gameStateJson
}

