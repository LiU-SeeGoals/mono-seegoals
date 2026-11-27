package plan

import (
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

type plannerCore struct {
	team            info.Team
	start           time.Time
	ActivityHandler ai.ActivityHandler
	Active          bool
}
