package session

import "time"

// IdleDetector
type IdleDetector struct {
	timer     *time.Timer
	untilIdle time.Duration
}

func NewIdleDetector(untilIdle time.Duration, onIdle func()) *IdleDetector {
	d := &IdleDetector{
		timer:     time.AfterFunc(untilIdle, onIdle),
		untilIdle: untilIdle,
	}

	d.timer.Stop()

	return d
}

func (d *IdleDetector) Touch() {
	d.timer.Reset(d.untilIdle)
}
