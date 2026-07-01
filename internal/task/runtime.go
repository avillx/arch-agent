package task

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrAlreadyRun = errors.New("task already run")
var ErrTaskIsNotRunning = errors.New("task is not running")

type RunningTask struct {
	*TaskConfig
	onExecute func(context.Context, *TaskConfig)
	done      chan struct{}
	stopOnce  sync.Once
	cron      Cron
}

// blocking
func (t *RunningTask) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.done:
			return
		case <-time.After(t.cron.NextTime()):
			t.onExecute(ctx, t.TaskConfig)
			if t.TaskConfig.oneshot {
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

func (r *TaskRuntime) Spawn(t *TaskConfig, cron Cron, exec func(context.Context, *TaskConfig)) {
	r.mu.Lock()

	if runningTask, ok := r.runningTasks[t.name]; ok {
		runningTask.Stop()
	}

	runningTask := &RunningTask{
		TaskConfig: t,
		onExecute:  exec,
		done:       make(chan struct{}),
		cron:       cron,
	}
	r.runningTasks[t.name] = runningTask
	r.mu.Unlock()

	go func() {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		runningTask.Start(ctx)

		r.mu.Lock()
		if runningTask, ok := r.runningTasks[t.name]; ok {
			runningTask.Stop()
			delete(r.runningTasks, t.name)
		}
		r.mu.Unlock()

		r.done <- t.name
	}()

}

func (r *TaskRuntime) Kill(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if runningTask, ok := r.runningTasks[name]; ok {
		runningTask.Stop()
		return nil
	}

	return ErrTaskIsNotRunning
}

func (r *TaskRuntime) DoneChannel() chan string {
	return r.done
}
