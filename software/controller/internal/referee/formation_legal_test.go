package referee

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
	"github.com/LiU-SeeGoals/controller/internal/rules"
)

func TestKickoffSlotsStayOutOfDefenseAreas(t *testing.T) {
	for _, bluePositive := range []bool{false, true} {
		gi := newFreeKickDefenseGameInfo(bluePositive)
		geometry, _ := gi.FieldGeometry()
		for _, slots := range [][]info.Position{
			kickoffDefenseSlots(gi, info.Blue, 7),
			kickoffSupportSlots(gi, info.Blue, 7),
		} {
			for _, slot := range slots {
				if insideInflatedDefenseArea(slot, geometry, -1) || insideInflatedDefenseArea(slot, geometry, 1) {
					t.Fatalf("blue+%v: slot %+v inside a defense area", bluePositive, slot)
				}
			}
		}
	}
}

func TestFreeKickSupportSlotsKeepClearOfBall(t *testing.T) {
	gi := newFreeKickDefenseGameInfo(false)
	for _, ball := range []info.Position{{}, {X: 1500, Y: -1600}, {X: 2025, Y: 0}} {
		for _, slot := range freeKickSupportSlots(gi, info.Blue, ball, 7) {
			if slot.Dist2d(ball) < rules.RestartBallKeepout {
				t.Fatalf("ball %+v: support slot %+v too close", ball, slot)
			}
		}
	}
}

func TestStopKickoffPrepareStaysOutsideStopKeepout(t *testing.T) {
	gi := newFreeKickDefenseGameInfo(false)
	pos := kickoffPreparePosition(gi, info.Blue, rules.FreeKickPrepareDistance)
	if pos.Dist2d(info.Position{}) < rules.StopBallKeepout {
		t.Fatalf("prepare position %+v inside STOP keep-out", pos)
	}
}
