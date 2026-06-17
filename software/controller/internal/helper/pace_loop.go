package helper

import (
	// "fmt"
	"time"
)

const (
	// PlannerLoopPeriod is the default target period for slow-brain planner ticks.
	PlannerLoopPeriod = 40 * time.Millisecond
	// ExecutorLoopPeriod is the default target period for the activity executor (fast brain).
	ExecutorLoopPeriod = 20 * time.Millisecond
)

// PaceLoop sleeps until period has elapsed since start. If work already took longer than period,
// it skips sleep. If name is non-empty, it prints a budget hint tagged with that loop name.
func PaceLoop(start time.Time, period time.Duration, name string) {
	elapsed := time.Since(start)
	if elapsed >= period {
		if name != "" {
			// fmt.Printf("%s: control loop ran too long (took %v, budget %v); use a faster computer or speed up this code path\n",
			// 	name, elapsed, period)
		}
		return
	}
	time.Sleep(period - elapsed)
}
