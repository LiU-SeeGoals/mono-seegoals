package helper

import "testing"

func TestNBSendKeepsNewestWhenFull(t *testing.T) {
	c := make(chan *int, 2)
	for i := 1; i <= 5; i++ {
		v := i
		NB_Send(c, &v)
	}
	if got := *<-c; got != 4 {
		t.Fatalf("first queued value = %d, want 4", got)
	}
	if got := *<-c; got != 5 {
		t.Fatalf("second queued value = %d, want 5", got)
	}
}
