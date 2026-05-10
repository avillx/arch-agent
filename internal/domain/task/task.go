package task

import (
	"context"
	"sync"
	"time"
)

// cron

// once at

type Reglament interface {
	NextTime() time.Duration
}

type Task struct {
	reglament   Reglament
	Description string
	executor    func(context.Context)
	stop        chan struct{}
	reset       chan struct{}
	once        sync.Once
}

func NewTask(
	reglament Reglament,
	Description string,
	executor func(context.Context),
) *Task {
	return &Task{
		reglament:   reglament,
		Description: Description,
		stop:        make(chan struct{}),
		reset:       make(chan struct{}),
		executor:    executor,
	}
}

func (t *Task) Stop() {
	t.once.Do(func() {
		close(t.stop)
	})
}

func (t *Task) SetReglament(reglament Reglament) {
	t.reglament = reglament
	t.reset <- struct{}{}
}

func (t *Task) Run(ctx context.Context) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	timer := time.NewTimer(t.reglament.NextTime())
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			t.executor(ctx)
			timer.Reset(t.reglament.NextTime())
		case <-ctx.Done():
			return ctx.Err()
		case <-t.reset:
			timer.Reset(t.reglament.NextTime())
		case <-t.stop:
			return nil
		}

	}
}
