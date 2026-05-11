package task

import (
	"context"
	"sync"
	"time"
)

type Reglament interface {
	NextTime() time.Duration
}

type Task struct {
	reglament Reglament
	// Description of the task
	Description string

	// call back on execution
	onCall func(context.Context, *Task)

	// channels for task handling
	stopFn func()
	mu     sync.Mutex
}

func NewTask(
	reglament Reglament,
	Description string,
	onCall func(context.Context, *Task),
) *Task {
	return &Task{
		reglament:   reglament,
		Description: Description,
		onCall:      onCall,
	}
}

func (t *Task) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopFn != nil {
		t.stopFn()
	}
}

// blocking
func (t *Task) Run(ctx context.Context) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	t.mu.Lock()
	t.stopFn = cancel
	t.mu.Unlock()

	for {
		timer := time.NewTimer(t.reglament.NextTime())

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()

		case <-timer.C:
			timer.Stop()

			if t.onCall != nil {
				t.onCall(ctx, t)
			}

			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			default:
			}

		}
	}
}
