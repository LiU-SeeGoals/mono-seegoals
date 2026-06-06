package debugstate

import (
	"sort"
	"sync"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

type RobotRoleDTO struct {
	Team int    `json:"Team"`
	Id   int    `json:"Id"`
	Role string `json:"Role"`
}

var robotRoles = struct {
	sync.RWMutex
	byTeam map[info.Team]map[info.ID]string
}{
	byTeam: make(map[info.Team]map[info.ID]string),
}

func SetRobotRole(team info.Team, id info.ID, role string) {
	robotRoles.Lock()
	defer robotRoles.Unlock()

	if _, ok := robotRoles.byTeam[team]; !ok {
		robotRoles.byTeam[team] = make(map[info.ID]string)
	}
	robotRoles.byTeam[team][id] = role
}

func ClearRobotRole(team info.Team, id info.ID) {
	robotRoles.Lock()
	defer robotRoles.Unlock()

	if _, ok := robotRoles.byTeam[team]; !ok {
		return
	}
	delete(robotRoles.byTeam[team], id)
}

func GetRobotRole(team info.Team, id info.ID) string {
	robotRoles.RLock()
	defer robotRoles.RUnlock()

	teamRoles, ok := robotRoles.byTeam[team]
	if !ok {
		return ""
	}
	return teamRoles[id]
}

func SnapshotRobotRoles() []RobotRoleDTO {
	robotRoles.RLock()
	defer robotRoles.RUnlock()

	roles := []RobotRoleDTO{}
	for team, teamRoles := range robotRoles.byTeam {
		for id, role := range teamRoles {
			if role == "" {
				continue
			}
			roles = append(roles, RobotRoleDTO{
				Team: int(team),
				Id:   int(id),
				Role: role,
			})
		}
	}

	sort.Slice(roles, func(i, j int) bool {
		if roles[i].Team == roles[j].Team {
			return roles[i].Id < roles[j].Id
		}
		return roles[i].Team < roles[j].Team
	})

	return roles
}
