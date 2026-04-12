package helper

import (
	"fmt"
	"time"
)

const (
	// PlannerLoopPeriod is the default target period for slow-brain planner ticks.
	PlannerLoopPeriod = 50 * time.Millisecond
	// ExecutorLoopPeriod is the default target period for the activity executor (fast brain).
	ExecutorLoopPeriod = 10 * time.Millisecond
)

// PaceLoop sleeps until period has elapsed since start. If work already took longer than period,
// it skips sleep. If optionalName is passed and non-empty, it also prints a budget warning using that label.
func PaceLoop(start time.Time, period time.Duration, optionalName ...string) {
	elapsed := time.Since(start)
	if elapsed >= period {
		if len(optionalName) > 0 && optionalName[0] != "" {
			fmt.Printf("%s: tick took %v (budget %v); machine too slow to keep pace\n", optionalName[0], elapsed, period)
		}
		return
	}
	time.Sleep(period - elapsed)
}
