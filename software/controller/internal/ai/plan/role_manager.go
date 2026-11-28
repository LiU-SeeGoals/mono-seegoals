package plan

import (
	"sort"
	"time"

	"github.com/LiU-SeeGoals/controller/internal/ai"
	"github.com/LiU-SeeGoals/controller/internal/debugstate"
	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/roles"
)

// roleSwitchMinDuration keeps a robot in its slot this long before the
// planner may move it to another one.
const roleSwitchMinDuration = 750 * time.Millisecond

type roleKind string

const (
	roleOffense roleKind = "offense"
	roleDefense roleKind = "defense"
)

// combinedRoleManager owns the offense/defense role objects of the field
// robots and maps tactical slots onto them.
type combinedRoleManager struct {
	activityHandler *ai.ActivityHandler
	gi              *info.GameInfo
	team            info.Team
	attackers       map[info.ID]*roles.OffenseRole
	defenders       map[info.ID]*roles.DefenseRole
	roleByRobot     map[info.ID]roleKind
	slotByRobot     map[info.ID]tacticalSlotKind
	lastChanged     map[info.ID]time.Time
}

func newCombinedRoleManager(activityHandler *ai.ActivityHandler, gi *info.GameInfo, team info.Team) *combinedRoleManager {
	return &combinedRoleManager{
		activityHandler: activityHandler,
		gi:              gi,
		team:            team,
		attackers:       make(map[info.ID]*roles.OffenseRole),
		defenders:       make(map[info.ID]*roles.DefenseRole),
		roleByRobot:     make(map[info.ID]roleKind),
		slotByRobot:     make(map[info.ID]tacticalSlotKind),
		lastChanged:     make(map[info.ID]time.Time),
	}
}

func (rm *combinedRoleManager) applySlotAssignments(assignments map[info.ID]tacticalSlotKind, now time.Time) {
	assignments = rm.reserveRetainedSlots(assignments, now)
	desired := make(map[info.ID]roleKind)
	for id, slot := range assignments {
		role := slot.roleKind()
		if role == "" {
			continue
		}
		desired[id] = role
	}

	for id := range rm.roleByRobot {
		if _, ok := desired[id]; !ok {
			rm.remove(id)
		}
	}

	for id, role := range desired {
		current, exists := rm.roleByRobot[id]
		currentSlot, hasSlot := rm.slotByRobot[id]
		nextSlot := assignments[id]
		if exists && current == role && hasSlot && currentSlot == nextSlot {
			continue
		}

		if !exists || current != role {
			switch role {
			case roleOffense:
				rm.assignOffense(id, now)
			case roleDefense:
				rm.assignDefense(id, now)
			}
		}

		rm.setSlot(id, nextSlot, now)
	}
}

// Reserve cooldown-protected slots before allocating the remaining slots. A
// swap must not give a slot to one robot while its previous owner still holds it.
func (rm *combinedRoleManager) reserveRetainedSlots(desired map[info.ID]tacticalSlotKind, now time.Time) map[info.ID]tacticalSlotKind {
	remaining := make(map[tacticalSlotKind]int)
	ids := make([]info.ID, 0, len(desired))
	for id, slot := range desired {
		remaining[slot]++
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	resolved := make(map[info.ID]tacticalSlotKind, len(desired))
	for _, id := range ids {
		current, exists := rm.slotByRobot[id]
		if exists && now.Sub(rm.lastChanged[id]) < roleSwitchMinDuration && remaining[current] > 0 {
			resolved[id] = current
			remaining[current]--
		}
	}
	for _, id := range ids {
		if _, reserved := resolved[id]; !reserved && remaining[desired[id]] > 0 {
			resolved[id] = desired[id]
			remaining[desired[id]]--
		}
	}
	slots := make([]tacticalSlotKind, 0, len(remaining))
	for slot := range remaining {
		slots = append(slots, slot)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, id := range ids {
		if _, assigned := resolved[id]; assigned {
			continue
		}
		for _, slot := range slots {
			if remaining[slot] > 0 {
				resolved[id] = slot
				remaining[slot]--
				break
			}
		}
	}
	return resolved
}

func (rm *combinedRoleManager) assignOffense(id info.ID, now time.Time) {
	rm.remove(id)
	attacker := roles.NewOffenseRole(id, *rm.activityHandler, rm.gi, rm.team)
	attacker.Init()
	rm.attackers[id] = attacker
	rm.roleByRobot[id] = roleOffense
	rm.lastChanged[id] = now
}

func (rm *combinedRoleManager) assignDefense(id info.ID, now time.Time) {
	rm.remove(id)
	defender := roles.NewDefenseRole(id, *rm.activityHandler, rm.gi, rm.team)
	defender.Init()
	rm.defenders[id] = defender
	rm.roleByRobot[id] = roleDefense
	rm.lastChanged[id] = now
}

func (rm *combinedRoleManager) remove(id info.ID) {
	delete(rm.attackers, id)
	delete(rm.defenders, id)
	delete(rm.roleByRobot, id)
	delete(rm.slotByRobot, id)
	rm.activityHandler.ClearActivity(id)
	debugstate.ClearRobotRole(rm.team, id)
}

func (rm *combinedRoleManager) setSlot(id info.ID, slot tacticalSlotKind, now time.Time) {
	rm.slotByRobot[id] = slot
	rm.lastChanged[id] = now
	debugstate.SetRobotRole(rm.team, id, slot.label())

	switch slot {
	case tacticalSlotBallChaser:
		if attacker, ok := rm.attackers[id]; ok {
			attacker.SetSlot(roles.OffenseSlot{Kind: roles.OffenseRoleChaser})
		}
	case tacticalSlotSupportShooter:
		if attacker, ok := rm.attackers[id]; ok {
			attacker.SetSlot(roles.OffenseSlot{Kind: roles.OffenseRoleShooter})
		}
	case tacticalSlotDefenderWall:
		if defender, ok := rm.defenders[id]; ok {
			defender.SetSlot(roles.DefenseSlot{Kind: roles.DefenseRoleWall})
		}
	case tacticalSlotDefenderHigh:
		if defender, ok := rm.defenders[id]; ok {
			defender.SetSlot(roles.DefenseSlot{Kind: roles.DefenseRoleHigh})
		}
	case tacticalSlotDefenderLow:
		if defender, ok := rm.defenders[id]; ok {
			defender.SetSlot(roles.DefenseSlot{Kind: roles.DefenseRoleLow})
		}
	}
}

func (rm *combinedRoleManager) configureSupportShooters(supportIDs []info.ID) {
	for index, id := range supportIDs {
		if attacker, ok := rm.attackers[id]; ok {
			attacker.SetSlot(roles.OffenseSlot{
				Kind:  roles.OffenseRoleShooter,
				Index: index,
				Count: len(supportIDs),
			})
		}
	}
}

func (rm *combinedRoleManager) offenseIDs() []info.ID {
	ids := make([]info.ID, 0, len(rm.attackers))
	for id := range rm.attackers {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func (rm *combinedRoleManager) defenseIDs() []info.ID {
	ids := make([]info.ID, 0, len(rm.defenders))
	for id := range rm.defenders {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func (rm *combinedRoleManager) idsForSlot(slot tacticalSlotKind) []info.ID {
	ids := []info.ID{}
	for id, assignedSlot := range rm.slotByRobot {
		if assignedSlot == slot {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func (rm *combinedRoleManager) idForSlotExcluding(
	slot tacticalSlotKind,
	excludedID info.ID,
	hasExcludedID bool,
) (info.ID, bool) {
	for _, id := range rm.idsForSlot(slot) {
		if !hasExcludedID || id != excludedID {
			return id, true
		}
	}
	return 0, false
}

func (rm *combinedRoleManager) configureOffenseReceivers(receiverIDs []info.ID) {
	for _, attacker := range rm.attackers {
		attacker.SetPassReceivers(receiverIDs)
	}
}
