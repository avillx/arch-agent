package task

import (
	"arch-agent/internal/agent"
	"sync"
	"time"
)

type Cron interface {
	NextTime() time.Duration
	Expression() string
}

type Task struct {
	Name        string
	Recipients  []agent.ID
	Description string
	Request     string
	Reglament   Cron
	OneShot     bool
}

func NewTask(
	name string,
	description string,
	recipients []agent.ID,
	request string,
	reglament Cron,
	oneShot bool,
) Task {
	return Task{
		Name:        name,
		Description: description,
		Recipients:  recipients,
		Request:     request,
		Reglament:   reglament,
		OneShot:     oneShot,
	}
}

type RunningTask struct {
	Task
	onExecute func(t Task)
	done      chan struct{}
	stopOnce  sync.Once
}

func NewRunningTask(t Task, onExecute func(t Task)) *RunningTask {
	return &RunningTask{
		Task:      t,
		onExecute: onExecute,
		done:      make(chan struct{}),
	}
}

// blocking
func (t *RunningTask) Start() {
	for {
		select {
		case <-t.done:
			return
		case <-time.After(t.Task.Reglament.NextTime()):
			t.onExecute(t.Task)
			if t.Task.OneShot {
				return
			}
		}
	}
}

func (t *RunningTask) Stop() {
	t.stopOnce.Do(func() { close(t.done) })
}
