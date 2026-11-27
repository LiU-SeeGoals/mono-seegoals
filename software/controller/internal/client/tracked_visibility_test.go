package client

import (
	"testing"

	"github.com/LiU-SeeGoals/proto_go/ssl_vision"
	"google.golang.org/protobuf/proto"
)

func TestTrackedRobotVisible(t *testing.T) {
	cases := []struct {
		name       string
		visibility *float32
		want       bool
	}{
		{"omitted", nil, true},
		{"visible", proto.Float32(1), true},
		{"partly", proto.Float32(0.3), true},
		{"lost", proto.Float32(0), false},
	}
	for _, tc := range cases {
		robot := &ssl_vision.TrackedRobot{Visibility: tc.visibility}
		if got := trackedRobotVisible(robot); got != tc.want {
			t.Errorf("%s: trackedRobotVisible = %v, want %v", tc.name, got, tc.want)
		}
	}
}
