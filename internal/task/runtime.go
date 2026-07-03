package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	onStop    func() error
}

func newRunningTask(
	cfg *TaskConfig,
	cron Cron,
	onExecute func(context.Context, *TaskConfig),
	onStop func() error,
) (*RunningTask, error) {
	if cfg == nil {
		return nil, fmt.Errorf("task config must be non nil")
	}
	if cron == nil {
		return nil, fmt.Errorf("task %s: cron must be non nil", cfg.name)
	}
	if onExecute == nil {
		return nil, fmt.Errorf("task %s: onExecute must be non nil", cfg.name)
	}
	if onStop == nil {
		return nil, fmt.Errorf("task %s: onStop must be non nil", cfg.name)
	}

	return &RunningTask{
		TaskConfig: cfg,
		onExecute:  onExecute,
		done:       make(chan struct{}),
		cron:       cron,
		onStop:     onStop,
	}, nil
}

// blocking
func (t *RunningTask) start() {
	for {
		select {
		case <-t.done:
			return
		case <-time.After(t.cron.NextTime()):
			t.onExecute(context.Background(), t.TaskConfig)
			if t.TaskConfig.oneshot {
				return
			}
		}
	}
}

func (t *RunningTask) stop() {
	t.stopOnce.Do(func() {
		close(t.done)
		if err := t.onStop(); err != nil {
			slog.Error("running task: stopped with errors", "task", t.Name(), "error", err)
		}
	})
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

func (r *TaskRuntime) Start(rt *RunningTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.runningTasks[rt.name]; ok {
		return ErrAlreadyRun
	}
	r.runningTasks[rt.name] = rt

	go func() {

		// block until task stopped
		rt.start()

		r.mu.Lock()
		defer r.mu.Unlock()

		if runningTask, ok := r.runningTasks[rt.name]; ok {
			runningTask.stop()
			delete(r.runningTasks, rt.name)
		}

	}()

	return nil
}

func (r *TaskRuntime) Kill(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rt, ok := r.runningTasks[name]
	if !ok {
		return ErrTaskIsNotRunning
	}
	rt.stop()
	return nil
}
