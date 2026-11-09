package scripts

import (
	"fmt"
	"time"

	act "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/scripting"
)

func DriveSquare(s *scripting.Script) {
	const id = info.ID(3)
	const side = 1000.0
	const pause = time.Second
	const reached = 150.0
	start := info.Position{X: -2000, Y: -1000}

	corners := []info.Position{
		{X: start.X, Y: start.Y},
		{X: start.X + side, Y: start.Y},
		{X: start.X + side, Y: start.Y + side},
		{X: start.X, Y: start.Y + side},
	}
	// teleport ball out of the way
	s.TeleportBall(info.Position{X:10000, Y:10000})
	for lap := 1; lap <= 3; lap++ {
		for _, corner := range corners {
			legStart := time.Now()
			s.Do(act.NewMoveToPosition(s.Team(), id, corner))
			arrived := s.WaitUntil(func(*info.GameInfo) bool {
				pos, ok := s.Robot(id)
				return ok && pos.Dist2d(corner) < reached
			}, 10*time.Second)
			s.Stop(id)
			pos, _ := s.Robot(id)
			if !arrived {
				fmt.Printf("FAIL: lap %d, never reached (%.0f, %.0f), stuck %.0f mm away\n",
					lap, corner.X, corner.Y, pos.Dist2d(corner))
				return
			}
			fmt.Printf("lap %d: reached (%.0f, %.0f) in %v, off by %.0f mm\n",
				lap, corner.X, corner.Y, time.Since(legStart).Round(100*time.Millisecond), pos.Dist2d(corner))
			s.Wait(pause)
		}
	}
	fmt.Println("PASS: drove 3 squares")
}
