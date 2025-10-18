package info

type Zone struct {
	Scores []float64
	Score  float64
}

type RobotAnalysisTeam [TEAM_SIZE]*RobotAnalysis
type TeamAnalysis struct {
	Robots   RobotAnalysisTeam
	ZoneSize float64
	Zones    [][]Zone
}

type RobotAnalysis struct {
	active           bool
	id               ID
	position         Position
	destination      Position
	velocity         Position
	maxMoveSpeed     float64 // mm/s
	maxRotationSpeed float64 // rad/s
	acceleration     float64 // mm/s^2
	deceleration     float64 // mm/s^2
}

func (r *RobotAnalysis) IsActive() bool {
	return r.active
}

func (r *RobotAnalysis) GetID() ID {
	return r.id
}

func (r *RobotAnalysis) GetPosition() Position {
	return r.position
}

func (r *RobotAnalysis) GetDestination() Position {
	return r.destination
}

func (r *RobotAnalysis) GetVelocity() Position {
	return r.velocity
}

func (r *RobotAnalysis) GetMaxMoveSpeed() float64 {
	return r.maxMoveSpeed
}

func (r *RobotAnalysis) GetMaxRotationSpeed() float64 {
	return r.maxRotationSpeed
}

func (r *RobotAnalysis) GetAcceleration() float64 {
	return r.acceleration
}

func (r *RobotAnalysis) GetDeceleration() float64 {
	return r.deceleration
}

func (r *RobotAnalysis) SetDestination(destination *Position) {
	r.destination = *destination
}

type BallAnalysis struct {
	position    Position
	velocity    Position
	destination Position
}

func (b *BallAnalysis) GetPosition() Position {
	return b.position
}

func (b *BallAnalysis) GetVelocity() Position {
	return b.velocity
}

func (b *BallAnalysis) GetDestination() Position {
	return b.destination
}

func (b *BallAnalysis) SetDestination(destination *Position) {
	b.destination = *destination
}

type FieldInfo struct {
	Length float64
	Width  float64
}

type GameAnalysis struct {
	team      Team
	MyTeam    *TeamAnalysis
	OtherTeam *TeamAnalysis
	Ball      *BallAnalysis
	FieldInfo FieldInfo
}

func calMoveSpeed(robot *Robot) float64 {
	velocity := robot.GetVelocity()
	return velocity.Norm()
}

func calRotationSpeed(robot *Robot) float64 {
	velocity := robot.GetVelocity()
	return velocity.Angle
}

func calAcceleration(robot *Robot) float64 {
	acceleration := robot.GetAcceleration()
	return acceleration
}

func calDeceleration(robot *Robot) float64 {
	return -calAcceleration(robot)
}

func updateTeam(gameStateTeam *RobotTeam, teamAnalysis *TeamAnalysis) {
	for _, robot := range gameStateTeam {
		rAn := teamAnalysis.Robots[robot.GetID()]
		if robot.IsActive() {
			rAn.active = true
			rAn.id = robot.GetID()
			robotPos, _ := robot.GetPosition()
			rAn.position = robotPos
			rAn.velocity = robot.GetVelocity()

			if speed := calMoveSpeed(robot); speed > rAn.maxMoveSpeed {
				rAn.maxMoveSpeed = speed
			}
			if rotationSpeed := calRotationSpeed(robot); rotationSpeed > rAn.maxRotationSpeed {
				rAn.maxRotationSpeed = rotationSpeed
			}
			if acceleration := calAcceleration(robot); acceleration > rAn.acceleration {
				rAn.acceleration = acceleration
			}
			if deceleration := calDeceleration(robot); deceleration < rAn.deceleration {
				rAn.deceleration = deceleration
			}

		} else {
			rAn.active = false
		}
	}
}

func updateBall(gameStateBall *Ball, ballAnalysis *BallAnalysis) {
	ballPos, _ := gameStateBall.GetPosition()
	ballAnalysis.position = ballPos
	ballAnalysis.velocity = gameStateBall.GetVelocity()
}

// TODO: Implement this function
// func (an *GameAnalysis) updateBallDistances(gamestateObj *gameinfo.GameState) {
// 	// Reset the distances
// 	an.distancesToBall = []float64{}

// 	// Get the position of the ball
// 	ball := gamestateObj.Ball.GetPosition()
// 	ballX := ball.AtVec(0)
// 	ballY := ball.AtVec(1)

// 	// Calculate the distances of the robots to the ball, storing them in order based on the robot id
// 	for _, robot := range gamestateObj.GetTeam(an.team) {
// 		robotX := robot.GetPosition().AtVec(0)
// 		robotY := robot.GetPosition().AtVec(1)

// 		distance := float64(math.Sqrt(
// 			math.Pow(robotX - ballX, 2) +
// 			math.Pow(robotY - ballY, 2)))
// 		an.distancesToBall = append(an.distancesToBall, distance)
// 	}
// }

func NewTeamAnalysis(fieldLength, fieldWidth, zoneSize float64) *TeamAnalysis {
	teamAnalysis := TeamAnalysis{}
	teamAnalysis.Robots = [TEAM_SIZE]*RobotAnalysis{}
	var i ID
	for i = 0; i < TEAM_SIZE; i++ {
		teamAnalysis.Robots[i] = &RobotAnalysis{}
	}

	teamAnalysis.ZoneSize = zoneSize
	hight := int(fieldLength / zoneSize)
	width := int(fieldWidth / zoneSize)
	teamAnalysis.Zones = make([][]Zone, hight)

	// Initialize the Zones
	for i := 0; i < hight; i++ {
		teamAnalysis.Zones[i] = make([]Zone, width)
	}

	return &teamAnalysis
}

func NewGameAnalysis(fieldLength, fieldWidth, zoneSize float64, team Team) *GameAnalysis {
	analysis := GameAnalysis{}
	analysis.team = team
	analysis.MyTeam = NewTeamAnalysis(fieldLength, fieldWidth, zoneSize)
	analysis.OtherTeam = NewTeamAnalysis(fieldLength, fieldWidth, zoneSize)
	analysis.Ball = &BallAnalysis{}
	return &analysis
}

func (analysis *GameAnalysis) UpdateState(gameState *GameState) {
	updateTeam(gameState.GetTeam(analysis.team), analysis.MyTeam)
	updateTeam(gameState.GetOtherTeam(analysis.team), analysis.OtherTeam)
	updateBall(gameState.GetBall(), analysis.Ball)
}

func updateZone(team *TeamAnalysis, fieldInfo FieldInfo, zoneSize float64, scoringFunc func(x float64, y float64, robots RobotAnalysisTeam) float64) {
	// Update the zones
	for i := 0; i < len(team.Zones); i++ {
		for j := 0; j < len(team.Zones[i]); j++ {
			// middle of the playing field in 0,0 so the zone need to be adjusted to the correct position
			x := float64(i)*zoneSize - fieldInfo.Length/2 + zoneSize/2
			y := float64(j)*zoneSize - fieldInfo.Width/2 + zoneSize/2
			team.Zones[i][j].Score = scoringFunc(x, y, team.Robots)
		}
	}
}

func (analysis *GameAnalysis) UpdateMyZones(scoringFunc func(x float64, y float64, robots RobotAnalysisTeam) float64) {
	updateZone(analysis.MyTeam, analysis.FieldInfo, analysis.MyTeam.ZoneSize, scoringFunc)
}

func (analysis *GameAnalysis) UpdateOtherZones(scoringFunc func(x float64, y float64, robots RobotAnalysisTeam) float64) {
	updateZone(analysis.OtherTeam, analysis.FieldInfo, analysis.OtherTeam.ZoneSize, scoringFunc)
}
