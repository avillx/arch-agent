package task

import (
	"time"
)

// every
type Every struct {
	D time.Duration
}

func (e Every) NextTime() time.Duration {
	return e.D
}

// daily
type Daily struct {
	Hour, Minute int
}

func (d Daily) NextTime() time.Duration {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), d.Hour, d.Minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

// weekly
type Weekly struct {
	Weekday      time.Weekday
	Hour, Minute int
}

func (w Weekly) NextTime() time.Duration {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), w.Hour, w.Minute, 0, 0, now.Location())
	daysUntil := (int(w.Weekday) - int(now.Weekday()) + 7) % 7
	next = next.AddDate(0, 0, daysUntil)
	if !next.After(now) {
		next = next.AddDate(0, 0, 7)
	}
	return next.Sub(now)
}

// monthly
type Monthly struct {
	Day, Hour, Minute int
}

func (m Monthly) NextTime() time.Duration {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), m.Day, m.Hour, m.Minute, 0, 0, now.Location())
	if !next.After(now) {
		next = time.Date(now.Year(), now.Month()+1, m.Day, m.Hour, m.Minute, 0, 0, now.Location())
	}
	return next.Sub(now)
}
