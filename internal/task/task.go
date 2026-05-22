package task

import (
	"sync"
	"time"
)

type Recipient string

type Reglament interface {
	NextTime() time.Duration
	Type() string
	String() string
}

type Task struct {
	Name        string
	Recipients  []Recipient
	Description string
	Request     string
	Reglament   Reglament
	OneShot     bool
}

func NewTask(
	name string,
	description string,
	recipients []Recipient,
	request string,
	reglament Reglament,
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
