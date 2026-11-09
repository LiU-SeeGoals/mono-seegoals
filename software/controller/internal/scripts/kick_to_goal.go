package scripts

import (
	"fmt"
	"time"

	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/scripting"
)

func KickToGoal(s *scripting.Script) {
	robot := info.ID(3)

	s.PlaceBall(info.Position{X: 0, Y: 0})
	s.Place(robot, info.Position{X: -1000, Y: 0})

	if !s.Run(act.NewMoveToBall(s.Team(), robot), 10*time.Second) {
		fmt.Println("FAIL: robot never reached the ball")
		return
	}

	s.Do(act.NewKickAtPosition(s.Team(), robot, s.EnemyGoal()))
	scored := s.WaitUntil(func(*info.GameInfo) bool { return s.BallInEnemyGoal() }, 8*time.Second)

	if scored {
		fmt.Println("PASS: ball in goal")
	} else {
		fmt.Println("FAIL: no goal within 8 s")
	}
}
