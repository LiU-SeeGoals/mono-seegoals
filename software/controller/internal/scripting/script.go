package scripting

import (
	"fmt"
	"math"
	"runtime/debug"
	"sync"
	"time"

	coreai "github.com/LiU-SeeGoals/controller/internal/ai"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

const (
	placeTolerance = 150.0
	placeRetry     = 500 * time.Millisecond
	placeSettle    = 300 * time.Millisecond
	pauseWarnDelay = time.Second
)

type Teleporter interface {
	TeleportRobot(x, y float32, id uint32, team info.Team)
	TeleportBall(x, y float32)
}

type Planner interface {
	Init(activities *[info.TEAM_SIZE]act.Activity, lock *sync.Mutex, team info.Team)
	Tick(gi *info.GameInfo)
	Kill()
}

type Script struct {
	team        info.Team
	gi          *info.GameInfo
	handler     coreai.ActivityHandler
	sim         Teleporter
	planner     Planner
	fn          func(*Script)
	frame       chan struct{}
	yield       chan struct{}
	active      bool
	started     bool
	finished    bool
	paused      string
	pausedSince time.Time
	warned      bool
}

func New(team info.Team, sim Teleporter, fn func(*Script)) *Script {
	return &Script{
		team:  team,
		sim:   sim,
		fn:    fn,
		frame: make(chan struct{}),
		yield: make(chan struct{}),
	}
}

func (s *Script) Init(activities *[info.TEAM_SIZE]act.Activity, lock *sync.Mutex, team info.Team) {
	s.handler.Activities = activities
	s.handler.Activity_lock = lock
	s.team = team
	s.active = true
}

func (s *Script) Tick(gi *info.GameInfo) {
	if !s.active {
		return
	}
	s.gi = gi
	s.warnIfPaused()
	if s.finished {
		s.StopAll()
		return
	}
	if s.planner != nil {
		s.planner.Tick(gi)
	}
	if !s.started {
		s.started = true
		go s.run()
	}
	s.frame <- struct{}{}
	<-s.yield
}

func (s *Script) warnIfPaused() {
	paused := ""
	if s.gi.Status != nil && s.gi.Status.GetGameEvent() != nil {
		switch s.gi.Status.GetGameEvent().GetCurrentState() {
		case info.STATE_HALTED:
			paused = "HALT"
		case info.STATE_STOPPED:
			paused = "STOP"
		}
	}
	if paused != s.paused {
		s.paused, s.pausedSince, s.warned = paused, time.Now(), false
	}
	if paused != "" && !s.warned && time.Since(s.pausedSince) > pauseWarnDelay {
		fmt.Printf("game controller is in %s, robots will not drive normally; press Force Start\n", paused)
		s.warned = true
	}
}

func (s *Script) Kill() {
	s.active = false
}

func (s *Script) run() {
	<-s.frame
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("script panicked: %v\n%s", r, debug.Stack())
			Logger.Errorf("script panicked: %v", r)
		} else {
			fmt.Println("script finished")
		}
		s.UsePlanner(nil)
		s.finished = true
		s.StopAll()
		s.yield <- struct{}{}
	}()
	s.fn(s)
}

func (s *Script) Team() info.Team {
	return s.team
}

func (s *Script) Game() *info.GameInfo {
	return s.gi
}

func (s *Script) NextFrame() {
	s.yield <- struct{}{}
	<-s.frame
}

func (s *Script) Wait(d time.Duration) {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		s.NextFrame()
	}
}

func (s *Script) WaitUntil(cond func(*info.GameInfo) bool, timeout time.Duration) bool {
	end := time.Now().Add(timeout)
	for {
		if cond(s.gi) {
			return true
		}
		if !time.Now().Before(end) {
			return false
		}
		s.NextFrame()
	}
}

func (s *Script) UsePlanner(p Planner) {
	if s.planner != nil {
		s.planner.Kill()
	}
	s.planner = p
	if p != nil {
		p.Init(s.handler.Activities, s.handler.Activity_lock, s.team)
	}
}

func (s *Script) Do(a act.Activity) {
	s.handler.AddActivity(a)
}

func (s *Script) Run(a act.Activity, timeout time.Duration) bool {
	s.Do(a)
	if s.WaitUntil(a.Achieved, timeout) {
		return true
	}
	fmt.Printf("%v did not finish within %v\n", a, timeout)
	return false
}

func (s *Script) Stop(id info.ID) {
	s.Do(act.NewStop(id))
}

func (s *Script) StopAll() {
	for id := info.ID(0); id < info.TEAM_SIZE; id++ {
		s.Stop(id)
	}
}

func (s *Script) Robot(id info.ID) (info.Position, bool) {
	pos, err := s.gi.State.GetRobotPosition(s.team, id)
	return pos, err == nil
}

func (s *Script) Ball() (info.Position, bool) {
	pos, err := s.gi.State.GetBall().GetEstimatedPosition()
	return pos, err == nil
}

func (s *Script) EnemyGoal() info.Position {
	return s.gi.EnemyGoalCenter(s.team)
}

func (s *Script) BallInEnemyGoal() bool {
	geometry, ok := s.gi.FieldGeometry()
	ball, seen := s.Ball()
	if !ok || !seen {
		return false
	}
	goalX := -s.gi.OwnHalfXSign(s.team) * geometry.Length / 2
	return ball.X*math.Copysign(1, goalX) > math.Abs(goalX) && math.Abs(ball.Y) < geometry.GoalWidth/2
}

func (s *Script) Place(id info.ID, pos info.Position) {
	s.Stop(id)
	s.waitPlaced(fmt.Sprintf("robot %d", id), pos,
		func() { s.sim.TeleportRobot(float32(pos.X), float32(pos.Y), uint32(id), s.team) },
		func() (info.Position, bool) { return s.Robot(id) },
	)
}

func (s *Script) PlaceBall(pos info.Position) {
	s.waitPlaced("ball", pos,
		func() { s.sim.TeleportBall(float32(pos.X/1000), float32(pos.Y/1000)) },
		s.Ball,
	)
}

func (s *Script) TeleportRobot(id info.ID, pos info.Position) {
	s.Stop(id)
	if s.sim == nil {
		fmt.Printf("Move robot %d to (%.0f, %.0f)\n", id, pos.X, pos.Y)
		return
	}
	s.sim.TeleportRobot(float32(pos.X), float32(pos.Y), uint32(id), s.team)
}

func (s *Script) TeleportBall(pos info.Position) {
	if s.sim == nil {
		fmt.Printf("Move ball to (%.0f, %.0f)\n", pos.X, pos.Y)
		return
	}
	s.sim.TeleportBall(float32(pos.X/1000), float32(pos.Y/1000))
}

func (s *Script) waitPlaced(name string, target info.Position, teleport func(), current func() (info.Position, bool)) {
	if s.sim == nil {
		fmt.Printf("Place %s at (%.0f, %.0f)\n", name, target.X, target.Y)
	}
	var lastTeleport, inPlaceSince time.Time
	for {
		now := time.Now()
		if pos, seen := current(); seen && pos.Dist2d(target) < placeTolerance {
			if inPlaceSince.IsZero() {
				inPlaceSince = now
			}
			if now.Sub(inPlaceSince) >= placeSettle {
				return
			}
		} else {
			inPlaceSince = time.Time{}
			if s.sim != nil && now.Sub(lastTeleport) >= placeRetry {
				teleport()
				lastTeleport = now
			}
		}
		s.NextFrame()
	}
}
