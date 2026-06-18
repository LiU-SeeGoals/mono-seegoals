package helper

import (
	"fmt"
	"time"
)

const frameSkipPrintPeriod = time.Second

type FrameSkipMonitor struct {
	name        string
	lastFrame   uint64
	skipped     uint64
	lastPrinted time.Time
}

func NewFrameSkipMonitor(name string) *FrameSkipMonitor {
	return &FrameSkipMonitor{name: name}
}

func (m *FrameSkipMonitor) Observe(frame uint64) {
	if frame == 0 {
		return
	}

	now := time.Now()
	if m.lastPrinted.IsZero() {
		m.lastPrinted = now
	}
	if m.lastFrame != 0 && frame > m.lastFrame+1 {
		m.skipped += frame - m.lastFrame - 1
	}
	m.lastFrame = frame

	if m.skipped == 0 || now.Sub(m.lastPrinted) < frameSkipPrintPeriod {
		return
	}

	fmt.Printf("%s: missed %d vision frames (latest frame %d)\n", m.name, m.skipped, frame)
	m.skipped = 0
	m.lastPrinted = now
}
