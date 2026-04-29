package ai

import (
	"sync"

	"github.com/LiU-SeeGoals/controller/internal/ai/pathplanner"
	"github.com/LiU-SeeGoals/controller/internal/info"
)

var pathServiceRegistry = struct {
	mu     sync.RWMutex
	byTeam map[info.Team]*pathplanner.Planner
}{
	byTeam: make(map[info.Team]*pathplanner.Planner),
}

// SetPathService registers the path planner for a team. Call from AI construction
// (one planner per team).
func SetPathService(team info.Team, p *pathplanner.Planner) {
	pathServiceRegistry.mu.Lock()
	defer pathServiceRegistry.mu.Unlock()
	pathServiceRegistry.byTeam[team] = p
}

func getPathService(team info.Team) *pathplanner.Planner {
	pathServiceRegistry.mu.RLock()
	defer pathServiceRegistry.mu.RUnlock()
	return pathServiceRegistry.byTeam[team]
}
