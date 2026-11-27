package ai

import (
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/action"
	"github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type stubActivity struct {
	id       info.ID
	achieved int
}

func (a *stubActivity) GetAction(*info.GameInfo) action.Action { return &action.Stop{Id: int(a.id)} }
func (a *stubActivity) Achieved(*info.GameInfo) bool           { a.achieved++; return false }
func (a *stubActivity) String() string                         { return fmt.Sprintf("stub %d", a.id) }
func (a *stubActivity) GetID() info.ID                         { return a.id }

// stubPlanner assigns activities to the given IDs on every tick.
type stubPlanner struct {
	handler ActivityHandler
	ids     []info.ID
	ticks   int
	acts    map[info.ID]*stubActivity
}

func (p *stubPlanner) Init(activities *[info.TEAM_SIZE]activity.Activity, lock *sync.Mutex, _ info.Team) {
	p.handler = ActivityHandler{Activities: activities, Activity_lock: lock}
	p.acts = map[info.ID]*stubActivity{}
}

func (p *stubPlanner) Tick(*info.GameInfo) {
	p.ticks++
	for _, id := range p.ids {
		if p.acts[id] == nil {
			p.acts[id] = &stubActivity{id: id}
		}
		p.handler.AddActivity(p.acts[id])
	}
}

func (p *stubPlanner) Kill() {}

func runningGameInfo() *info.GameInfo {
	gi := info.NewGameInfo(10)
	gi.Status.GetGameEvent().UpdateFromRefCommand(info.FORCE_START, 1, 0, 0, info.UNINITIALIZED, 0, false)
	return gi
}

func TestGetActionsIsSynchronousAndOrdered(t *testing.T) {
	before := runtime.NumGoroutine()
	planner := &stubPlanner{ids: []info.ID{5, 1, 3}}
	ai := NewAi(info.Blue, planner)
	gi := runningGameInfo()

	for frame := 1; frame <= 3; frame++ {
		actions := ai.GetActions(gi)
		if planner.ticks != frame {
			t.Fatalf("planner ticked %d times after %d frames", planner.ticks, frame)
		}
		if len(actions) != 3 {
			t.Fatalf("got %d actions, want 3", len(actions))
		}
		for i, want := range []int{1, 3, 5} {
			if got := actions[i].(*action.Stop).Id; got != want {
				t.Fatalf("action %d is for robot %d, want %d", i, got, want)
			}
		}
	}
	for id, act := range planner.acts {
		// Planners decide on achievement; the executor only asks for actions.
		if act.achieved != 0 {
			t.Fatalf("robot %d: executor called Achieved %d times, want 0", id, act.achieved)
		}
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Fatalf("NewAi/GetActions left %d background goroutines", after-before)
	}
}

func TestHotswapPlannerKeepsSharedActivities(t *testing.T) {
	first := &stubPlanner{ids: []info.ID{2}}
	second := &stubPlanner{ids: []info.ID{4}}
	ai := NewAi(info.Yellow, first)
	gi := runningGameInfo()
	ai.GetActions(gi)

	ai.HotswapPlanner(info.Yellow, second)
	second.handler.ClearActivities()
	actions := ai.GetActions(gi)
	if len(actions) != 1 || actions[0].(*action.Stop).Id != 4 {
		t.Fatalf("after hotswap got %v, want a single action for robot 4", actions)
	}
}

type continuingActivity struct {
	stubActivity
	from activity.Activity
}

func (c *continuingActivity) ContinueFrom(previous activity.Activity) { c.from = previous }

func TestAddActivityContinuesPreviousActivity(t *testing.T) {
	var activities [info.TEAM_SIZE]activity.Activity
	handler := &ActivityHandler{Activities: &activities, Activity_lock: &sync.Mutex{}}
	first := &continuingActivity{stubActivity: stubActivity{id: 4}}
	second := &continuingActivity{stubActivity: stubActivity{id: 4}}

	handler.AddActivity(first)
	handler.AddActivity(second)
	if first.from != nil || second.from != first {
		t.Fatal("AddActivity did not hand the previous activity to ContinueFrom")
	}
}
