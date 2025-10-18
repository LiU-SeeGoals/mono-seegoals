package ai

import (
	"sync"

	ai "github.com/LiU-SeeGoals/controller/internal/ai/activity"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type planner interface {
	run()
	Init(incoming <-chan info.GameInfo,
		activities *[info.TEAM_SIZE]ai.Activity,
		lock *sync.Mutex,
		team info.Team) bool
}
