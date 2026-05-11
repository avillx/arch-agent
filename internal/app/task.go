package service

import (
	"arch-agent/internal/domain/task"
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type TaskService struct {
	tasks map[string]*task.Task
	mu    sync.RWMutex
}

func NewTaskService() *TaskService {
	return &TaskService{
		tasks: map[string]*task.Task{},
	}
}

func (s *TaskService) Get(taskID string) (*task.Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[taskID]
	return task, ok
}

type TaskView struct {
	ID          string
	Description string
}

func (s *TaskService) AllTasks() []TaskView {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tvs := make([]TaskView, 0, len(s.tasks))
	for k, v := range s.tasks {
		tv := TaskView{
			ID:          k,
			Description: v.Description,
		}

		tvs = append(tvs, tv)
	}

	return tvs
}

func (s *TaskService) AddTask(ctx context.Context, id string, task *task.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, ok := s.tasks[id]; ok {
		task.Stop()
		slog.Info("task overwrited", "task", id)
	}

	s.tasks[id] = task

	go func() {
		if err := task.Run(ctx); err != nil {
			slog.Info("task closed", "task", id)
		}
		s.mu.Lock()
		delete(s.tasks, id)
		s.mu.Unlock()
	}()
}

func (s *TaskService) RemoveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; ok {
		delete(s.tasks, id)
		slog.Debug("task deleted", "task", id)
		return nil
	}

	return fmt.Errorf("tasks %s not found", id)
}
