package task

import (
	"arch-agent/internal/types"
	"context"
	"sync"
	"time"
)

type RunningTask struct {
	Task
	onExecute func(context.Context, Task)
	done      chan struct{}
	stopOnce  sync.Once
}

func NewRunningTask(t Task, onExecute func(context.Context, Task)) *RunningTask {
	return &RunningTask{
		Task:      t,
		onExecute: onExecute,
		done:      make(chan struct{}),
	}
}

// blocking
func (t *RunningTask) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.done:
			return
		case <-time.After(t.Task.Reglament.NextTime()):
			t.onExecute(ctx, t.Task)
			if t.Task.OneShot {
				return
			}
		}
	}
}

func (t *RunningTask) Stop() {
	t.stopOnce.Do(func() { close(t.done) })
}

// runtime
type TaskRuntime struct {
	runningTasks map[string]*RunningTask
	mu           sync.RWMutex
	done         chan string
}

func NewTaskRuntime() *TaskRuntime {
	return &TaskRuntime{
		runningTasks: map[string]*RunningTask{},
		done:         make(chan string),
	}
}

func (r *TaskRuntime) Spawn(t Task, exec func(context.Context, Task)) {
	r.mu.Lock()

	if runningTask, ok := r.runningTasks[t.Name]; ok {
		runningTask.Stop()
	}

	newTask := NewRunningTask(t, exec)
	r.runningTasks[t.Name] = newTask
	r.mu.Unlock()

	go func() {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		newTask.Start(ctx)

		r.mu.Lock()
		if runningTask, ok := r.runningTasks[t.Name]; ok {
			runningTask.Stop()
			delete(r.runningTasks, t.Name)
		}
		r.mu.Unlock()

		r.done <- t.Name
	}()
}

func (r *TaskRuntime) Kill(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if runningTask, ok := r.runningTasks[name]; ok {
		runningTask.Stop()
		return nil
	}

	return types.ErrIsNotExist
}

func (r *TaskRuntime) DoneChannel() chan string {
	return r.done
}
