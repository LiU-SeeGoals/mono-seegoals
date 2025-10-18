package ai

// import (
// 	"fmt"
// 	"math"
// 	"sync"
// 	"time"

// 	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
// 	"github.com/LiU-SeeGoals/controller/internal/info"
// 	"github.com/LiU-SeeGoals/controller/internal/logger"
// 	vis "github.com/LiU-SeeGoals/controller/internal/visualisation"
// 	"gonum.org/v1/gonum/mat"
// 	"gonum.org/v1/plot/plotter"
// )

// type plannerAo struct {
// 	plannerCore
// 	at_state int
// 	start    time.Time
// 	max_time time.Duration
// }

// func NewPlannerAo(team info.Team) *plannerAo {
// 	return &plannerAo{
// 		plannerCore: plannerCore{
// 			team: team,
// 		},
// 	}
// }

// func (m *plannerAo) Init(
// 	incoming <-chan info.GameInfo,
// 	activities *[info.TEAM_SIZE]ai.Activity,
// 	lock *sync.Mutex,
// 	team info.Team,
// ) {
// 	m.incomingGameInfo = incoming
// 	m.activities = activities // store pointer directly
// 	m.activity_lock = lock
// 	m.team = team
// 	m.start = time.Now()

// 	go m.run()
// }

// func (m *plannerAo) run() {

// 	gameInfo := <-m.incomingGameInfo
// 	fmt.Println(gameInfo.Status)

// 	// Basic idea
// 	// Defender: Get some dudes to guard the goal, stand in "line" formation towards ball
// 	//	- Function that returns indices for robots that should perform defense
// 	// Attacker: Chase ball, kick toward goal, turn to support when away from ball;
// 	// Support: Stand a bit away from attack so he can pass, turn into attacker when get ball
// 	robotPos := plotter.XYs{}

// 	vis.GetVisualiser().CreateEmptyNamedPlotWindow("raycast")
// 	for {
// 		// gameInfo.PrintField()
// 		// No need for slow brain to be fast
// 		time.Sleep(1 * time.Millisecond)

// 		// robots := []int{0,1,3}
// 		// if m.HandleRef(&gameInfo, robots) {
// 		// 	continue
// 		// }

//         // robot := robots[0]
// 		defenders := []info.ID{0,1}
// 		attackers := []info.ID{3}

// 		myRobotPos, err := gameInfo.State.GetTeam(m.team)[0].GetPosition()
// 		if err != nil {
// 			logger.Logger.Debugln("Big err")
// 		}

// 		robotPos = append(robotPos, plotter.XY{X: myRobotPos.X, Y: myRobotPos.Y})
// 		p := vis.ScatterPlt(robotPos)
// 		p.Title.Text = fmt.Sprintf("Robot %v team %v", 0, m.team)
// 		// fig.UpdatePlotWindow(p)

// 		m.defense(defenders)
// 		m.attack(attackers)
// 		m.rayMarch(attackers[0], gameInfo)
// 		// p := vis.LinePlt(plotRays)
// 		// fig.UpdatePlotWindow(p)
// 	}
// }

// func (m *plannerAo) defense(robots []info.ID){

// 	gi := <-m.incomingGameInfo

// 	var formation = map[info.ID][2]float64{
// 		robots[0]: {0, 0},
// 		robots[1]: {0, -200},
// 		// robots[2]: {0, 200},
// 	}

// 	def := gi.HomeGoalDefPos(m.team)
// 	ballpos, err := gi.State.GetBall().GetPosition()
// 	defY := ballpos.Y

// 	if err != nil {
// 		fmt.Println("Ball position is undefined")
// 	}

// 	defensePos := info.Position{X: def.X, Y: defY, Z: 0, Angle: def.Angle + math.Pi}

// 	for i := range robots {
// 		id := robots[i]
// 		offset := formation[id]
// 		// fmt.Printf("robots %v i %v id %v offest %v", robots, i, id, offset)
// 		formationPosx := defensePos.X + offset[0]
// 		formationPosy := defensePos.Y + offset[1]
// 		pos := info.Position{X: formationPosx, Y: formationPosy, Z: 0, Angle: defensePos.Angle}
// 		// fmt.Printf("Moving %v to %v\n", id, pos)
// 		m.AddActivity(ai.NewMoveToPosition(m.team, id, pos))
// 	}
// }

// func (m *plannerAo) attack(robots []info.ID){

// 	for i := range robots{
// 		// if m.activities[robots[i]] == nil {
// 			activityLoop := []ai.Activity{
// 				ai.NewMoveToBall(m.team, robots[i]),
// 				ai.NewKickTheBall(m.team, robots[i], info.Position{X: 2000, Y: 2000, Z: 0, Angle: 0}),
// 				// ai.NewKickToPlayer(m.team, 0, 1),
// 			}
// 			loop := ai.NewActivityLoop(robots[i], activityLoop)
// 			m.AddActivity(loop)
// 		// }
// 	}
// }

// func RadToDeg(rad float64) float64 {
//     return rad * (180.0 / math.Pi)
// }

// func DegToRad(deg float64) float64 {
//     return deg * (math.Pi / 180.0)
// }

// func (m *plannerAo) rayMarch(robot info.ID, gi info.GameInfo) {

// 	pos, err := gi.State.GetTeam(m.team)[robot].GetPosition()
// 	if err != nil {
// 		logger.Logger.Debugln("Robot pos not found")
// 		return
// 	}
// 	// enemy := gi.EnemyGoalLine(m.team)

// 	// Step size of i is the resolution of the rays

// 	angularRes := 10
// 	var rays []*mat.VecDense
// 	plotRays := plotter.XYs{}

// 	// TODO: Plot rays by creating line from robot pos to length of ray

// 	// Ray length in mm
// 	rayLen := 2000.0

// 	for angle := 0; angle < 360; angle+=angularRes {

// 		rad := DegToRad(float64(angle))

// 		dx := pos.X + rayLen * math.Cos(rad)
// 		dy := pos.Y + rayLen * math.Sin(rad)

// 		rayx := dx
// 		rayy := dy

// 		rays = append(rays, mat.NewVecDense(2, []float64{rayx, rayy}))
// 		plotRays = append(plotRays, plotter.XY{X: dx, Y: dy})
// 	}

// 	// Stepsize along ray in millimeter

// 	// stepSize := 20
// 	//
// 	// for i := 0; i < len(rays); i += 1{
// 	// 	for step := 0; step < int(rayLen) / stepSize; step += stepSize{
// 	// 		x :=
// 	// 		y :=
// 	// 	}
// 	// }

// 	p := vis.RayPlt(plotter.XY{X: pos.X, Y: pos.Y}, plotRays)
// 	fieldSize := gi.FieldSize()
// 	p.X.Min = -fieldSize.X/2
// 	p.X.Max = fieldSize.X/2

// 	p.Y.Min = -fieldSize.Y/2
// 	p.Y.Max = fieldSize.Y/2

// 	// p := vis.LinePlt(plotRays)
// 	vis.GetVisualiser().GetPlot("raycast").UpdatePlotWindow(p)
// 	// return plotRays
// }
