package ai

import (
	"time"
	. "github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type plannerCore struct {
	team             info.Team
	incomingGameInfo <-chan info.GameInfo
	start            time.Time
	ActivityHandler  ActivityHandler
}

