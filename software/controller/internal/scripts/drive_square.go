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

	corners := []info.Position{
		{X: 0, Y: 0},
		{X: side, Y: 0},
		{X: side, Y: side},
		{X: 0, Y: side},
	}
	// teleport ball out of the way 
	s.TeleportBall(info.Position{X:10000, Y:10000})
	for lap := 1; lap <= 3; lap++ {
		for _, corner := range corners {
			start := time.Now()
			if !s.Run(act.NewMoveToPosition(s.Team(), id, corner), 10*time.Second) {
				fmt.Printf("FAIL: lap %d, never reached (%.0f, %.0f)\n", lap, corner.X, corner.Y)
				return
			}
			s.Stop(id)
			pos, _ := s.Robot(id)
			fmt.Printf("lap %d: reached (%.0f, %.0f) in %v, off by %.0f mm\n",
				lap, corner.X, corner.Y, time.Since(start).Round(100*time.Millisecond), pos.Dist2d(corner))
			s.Wait(pause)
		}
	}
	fmt.Println("PASS: drove 3 squares")
}
