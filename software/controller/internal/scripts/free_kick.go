package scripts

import (
	"fmt"
	"time"

	plan "github.com/LiU-SeeGoals/controller/internal/ai/plan"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/referee"
	"github.com/LiU-SeeGoals/controller/internal/scripting"
)

func FreeKick(s *scripting.Script) {
	ball := info.Position{X: 1500, Y: -1000}

	s.PlaceBall(ball)
	s.Place(2, info.Position{X: 800, Y: -1000})
	s.Place(3, info.Position{X: 2500, Y: 800})
	s.Place(6, s.Game().HomeGoalDefPos(s.Team()))

	s.UsePlanner(plan.NewCombinedPlanWithRef(s.Team()))

	fmt.Println("In the game controller: press STOP, then FREE KICK for", s.Team())

	closestID, closest := info.ID(0), 1e9
	for state(s) != info.STATE_FREE_KICK {
		if state(s) == info.STATE_STOPPED {
			if pos, ok := s.Ball(); ok {
				if id, d := closestRobot(s, pos); d < closest {
					closestID, closest = id, d
				}
			}
		}
		s.NextFrame()
	}
	if closest < 500 {
		fmt.Printf("FAIL: robot %d came %.0f mm from the ball during STOP\n", closestID, closest)
	}

	kickFrom, _ := s.Ball()
	limit := referee.FreeKickMaxTime(s.Game().Status.GetDivision())
	start := time.Now()
	kicked := s.WaitUntil(func(*info.GameInfo) bool {
		pos, ok := s.Ball()
		return ok && pos.Dist2d(kickFrom) > 100
	}, limit)
	if !kicked {
		fmt.Printf("FAIL: ball not kicked within %v\n", limit)
		return
	}
	fmt.Printf("kick taken after %v\n", time.Since(start).Round(100*time.Millisecond))

	fouled := s.WaitUntil(func(*info.GameInfo) bool {
		return state(s) == info.STATE_STOPPED
	}, 3*time.Second)
	if fouled {
		fmt.Println("FAIL: auto-ref stopped the game after the kick (double touch?)")
		return
	}
	fmt.Println("PASS: clean free kick")
}

func state(s *scripting.Script) info.RefState {
	return s.Game().Status.GetGameEvent().GetCurrentState()
}

func closestRobot(s *scripting.Script, target info.Position) (info.ID, float64) {
	best, bestDist := info.ID(0), 1e9
	for id := info.ID(0); id < info.TEAM_SIZE; id++ {
		if pos, ok := s.Robot(id); ok && pos.Dist2d(target) < bestDist {
			best, bestDist = id, pos.Dist2d(target)
		}
	}
	return best, bestDist
}
