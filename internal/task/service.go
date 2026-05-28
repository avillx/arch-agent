package task

import (
	"context"
	"log/slog"
)

type TaskRecord struct {
	Active bool
	Task
}

type TaskRepo interface {
	All() (map[string]*TaskRecord, error)
	Get(id string) (*TaskRecord, error)
	Save(id string, t *TaskRecord) error
	Delete(id string) error
}

// service
type TaskService struct {
	runtime  *TaskRuntime
	executor *taskExecutor
	repo     TaskRepo
}

func NewTaskService(
	ctx context.Context,
	repo TaskRepo,
	executor *taskExecutor,
) (*TaskService, error) {

	s := &TaskService{
		repo:     repo,
		runtime:  NewTaskRuntime(),
		executor: executor,
	}

	s.processDoneTasks(ctx)

	recs, err := repo.All()
	if err != nil {
		return nil, err
	}

	for id, rec := range recs {
		if !rec.Active {
			continue
		}
		// TODO validate task on service creation
		// !Caution it may have edge case issues if agent is not exist
		// validation return error and program not started
		if err := s.Start(id); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *TaskService) All() (map[string]*TaskRecord, error) {
	return s.repo.All()
}

func (s *TaskService) New(t Task) (string, error) {

	// validate
	// if err := s.executor.Validate(t); err != nil {
	// 	return "", err
	// }

	// save to repo
	if err := s.repo.Save(t.Name, &TaskRecord{Active: false, Task: t}); err != nil {
		return "", err
	}

	return t.Name, nil
}

func (s *TaskService) Start(id string) error {

	rec, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	s.runtime.Spawn(rec.Task, s.executor.execute)
	rec.Active = true

	return s.repo.Save(id, rec)
}

func (s *TaskService) Stop(id string) error {
	return s.runtime.Kill(id)
}

func (s *TaskService) Delete(id string) error {
	s.runtime.Kill(id)

	return s.repo.Delete(id)
}

// blocking
func (s *TaskService) processDoneTasks(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case doneTaskID := <-s.runtime.DoneChannel():
				rec, err := s.repo.Get(doneTaskID)
				if err != nil {
					slog.Error("process done tasks", "task", doneTaskID, "error", err)
				}

				rec.Active = false

				if err := s.repo.Save(doneTaskID, rec); err != nil {
					slog.Error("process done tasks", "task", doneTaskID, "error", err)
				}
			}
		}
	}()
}
