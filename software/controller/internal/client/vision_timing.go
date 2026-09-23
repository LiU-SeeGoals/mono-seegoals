package client

import (
	"fmt"
	"os"
	"time"
)

var visionTimingEnabled = os.Getenv("SEEGOALS_VISION_TIMING") == "1"

// reportVisionTiming prints one sample per second for each source when enabled.
// These Unix timestamps need synchronized host clocks for an accurate age.
func reportVisionTiming(source string, capture, sent float64, last *time.Time) {
	if !visionTimingEnabled || capture < 1e9 {
		return
	}
	now := time.Now()
	if now.Sub(*last) < time.Second {
		return
	}
	*last = now
	arrival := float64(now.UnixNano()) / 1e9
	if sent >= capture {
		fmt.Printf("[vision-timing] %s age=%.1fms processor=%.1fms after-send=%.1fms\n",
			source, (arrival-capture)*1000, (sent-capture)*1000, (arrival-sent)*1000)
		return
	}
	fmt.Printf("[vision-timing] %s age=%.1fms\n", source, (arrival-capture)*1000)
}
