package service

import (
	"arch-agent/internal/domain/task"
	"arch-agent/internal/domain/types"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

type TaskService struct {
	tasks map[string]*task.Task
	mu    sync.RWMutex
}

func (s *TaskService) Task(id string) (*task.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if task, ok := s.tasks[id]; ok {
		return task, nil
	}

	return nil, errors.Join(types.ErrIsNotExist, fmt.Errorf("tasks %s not found", id))
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

func (s *TaskService) AddTask(ctx context.Context, id string, task *task.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; ok {
		return fmt.Errorf("task %s is already exist", id)
	}

	s.tasks[id] = task

	go func() {
		if err := task.Run(ctx); err != nil {
			slog.Error("task closed", "error", err)
		}
		delete(s.tasks, id)
	}()

	return nil
}

func (s *TaskService) RemoveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; ok {
		delete(s.tasks, id)
		return nil
	}

	return fmt.Errorf("tasks %s not found", id)
}
