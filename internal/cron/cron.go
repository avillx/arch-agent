package cron

import (
	"time"

	robcron "github.com/robfig/cron/v3"
)

type RobfigCron struct {
	expression string
	schedule   robcron.Schedule
}

func NewRobfigCron(expr string) (*RobfigCron, error) {

	parser := robcron.NewParser(
		robcron.Minute |
			robcron.Hour |
			robcron.Dom |
			robcron.Month |
			robcron.Dow,
	)

	sched, err := parser.Parse(expr)
	if err != nil {
		return nil, err
	}

	return &RobfigCron{
		schedule:   sched,
		expression: expr,
	}, nil
}

func (r *RobfigCron) NextTime() time.Duration {
	now := time.Now()
	next := r.schedule.Next(now)
	nextLocal := next.In(now.Location())
	return nextLocal.Sub(now)
}

func (r *RobfigCron) Expression() string {
	return r.expression
}
