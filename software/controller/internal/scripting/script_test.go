package scripting

import (
	"sync"
	"testing"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/action"
	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type fakeSim struct{ gi *info.GameInfo }

func (f *fakeSim) TeleportRobot(x, y float32, id uint32, team info.Team) {
	f.gi.State.SetBlueRobot(id, float64(x), float64(y), 0, time.Now().UnixMilli())
}

func (f *fakeSim) TeleportBall(x, y float32) {
	pos := info.Position{X: float64(x) * 1000, Y: float64(y) * 1000}
	f.gi.State.SetBall(pos.X, pos.Y, 0, time.Now().UnixMilli())
	f.gi.State.GetBall().SetEstimatedPosition(pos)
}

type countdown struct {
	id   info.ID
	left int
}

func (c *countdown) GetAction(*info.GameInfo) action.Action { return &action.Stop{Id: int(c.id)} }
func (c *countdown) Achieved(*info.GameInfo) bool           { c.left--; return c.left <= 0 }
func (c *countdown) String() string                         { return "countdown" }
func (c *countdown) GetID() info.ID                         { return c.id }

func TestScriptRunsInStepWithFramesAndStopsWhenDone(t *testing.T) {
	gi := info.NewGameInfo(2)
	activities := &[info.TEAM_SIZE]act.Activity{}
	var steps []string
	s := New(info.Blue, &fakeSim{gi}, func(s *Script) {
		s.PlaceBall(info.Position{X: 500})
		s.Place(3, info.Position{X: -1000})
		steps = append(steps, "placed")
		if s.Run(&countdown{id: 3, left: 3}, time.Second) {
			steps = append(steps, "ran")
		}
	})
	s.Init(activities, &sync.Mutex{}, info.Blue)

	deadline := time.Now().Add(3 * time.Second)
	for !s.finished && time.Now().Before(deadline) {
		s.Tick(gi)
		time.Sleep(5 * time.Millisecond)
	}
	if !s.finished {
		t.Fatalf("script did not finish, got steps %v", steps)
	}
	if len(steps) != 2 {
		t.Fatalf("got steps %v, want placed then ran", steps)
	}
	if pos, ok := s.Robot(3); !ok || pos.Dist2d(info.Position{X: -1000}) > 1 {
		t.Fatalf("robot 3 at %v (seen %v), want teleported to x=-1000", pos, ok)
	}
	if _, isStop := activities[3].(*act.Stop); !isStop {
		t.Fatalf("robot 3 activity after the script = %v, want Stop", activities[3])
	}
}

type countingPlanner struct {
	ticks  int
	killed bool
}

func (p *countingPlanner) Init(*[info.TEAM_SIZE]act.Activity, *sync.Mutex, info.Team) {}
func (p *countingPlanner) Tick(*info.GameInfo)                                        { p.ticks++ }
func (p *countingPlanner) Kill()                                                      { p.killed = true }

func TestUsePlannerTicksUntilScriptEnds(t *testing.T) {
	gi := info.NewGameInfo(2)
	planner := &countingPlanner{}
	s := New(info.Blue, nil, func(s *Script) {
		s.UsePlanner(planner)
		for i := 0; i < 5; i++ {
			s.NextFrame()
		}
	})
	s.Init(&[info.TEAM_SIZE]act.Activity{}, &sync.Mutex{}, info.Blue)

	for i := 0; i < 20 && !s.finished; i++ {
		s.Tick(gi)
	}
	if !s.finished {
		t.Fatal("script did not finish")
	}
	if planner.ticks != 5 || !planner.killed {
		t.Fatalf("planner ticked %d times (killed %v), want 5 ticks then killed", planner.ticks, planner.killed)
	}
}
