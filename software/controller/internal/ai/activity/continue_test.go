package activity

import (
	"testing"

	"github.com/LiU-SeeGoals/controller/internal/info"
)

func TestMoveToPositionContinuesLookAhead(t *testing.T) {
	prev := NewMoveToPosition(info.Blue, 2, info.Position{X: 1000})
	prev.stickyDest, prev.stickySet, prev.stickyPathID = info.Position{X: 300}, true, pathLookID{n: 4}

	next := NewMoveToPosition(info.Blue, 2, info.Position{X: 1010})
	next.ContinueFrom(prev)
	if !next.stickySet || next.stickyDest.X != 300 || next.stickyPathID.n != 4 {
		t.Fatal("rebuilt MoveToPosition lost its sticky look-ahead")
	}

	other := NewMoveToPosition(info.Blue, 3, info.Position{X: 1010})
	other.ContinueFrom(prev)
	if other.stickySet {
		t.Fatal("look-ahead carried over to another robot")
	}
}

func TestAlignBallContinuesImplicitLatchOnly(t *testing.T) {
	prev := NewAlign(info.Blue, 1, info.Position{X: 4500}, info.Position{})
	next := NewAlign(info.Blue, 1, info.Position{X: 4500}, info.Position{})
	next.ContinueFrom(prev)
	if next.latch != prev.latch {
		t.Fatal("rebuilt AlignBall did not keep its latch")
	}

	owned := &AlignLatch{}
	withOwn := NewAlign(info.Blue, 1, info.Position{X: 4500}, info.Position{})
	withOwn.SetLatch(owned)
	withOwn.ContinueFrom(prev)
	if withOwn.latch != owned {
		t.Fatal("ContinueFrom replaced a latch set with SetLatch")
	}

	after := NewAlign(info.Blue, 1, info.Position{X: 4500}, info.Position{})
	after.ContinueFrom(withOwn)
	if after.latch == owned {
		t.Fatal("a caller-owned latch leaked into an unmanaged AlignBall")
	}
}
