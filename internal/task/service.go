package task

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"context"
	"errors"
	"log/slog"
)

var ErrIsNotExist = types.ErrIsNotExist
var ErrAlreadyExist = types.ErrAlreadyExist
var ErrCron = errors.New("cron is not support this expression")

type TaskRepo interface {
	All() (map[string]*TaskRecord, error)
	Get(id string) (*TaskRecord, error)
	Delete(id string) error
	Save(t *TaskRecord) error
}

// service
type Service struct {
	runtime     *TaskRuntime
	executor    *executor
	cronFactory func(string) (Cron, error)
	repo        TaskRepo
}

func NewService(
	ctx context.Context,
	repo TaskRepo,
	cronFactory func(string) (Cron, error),
	executor *executor,
) (*Service, error) {

	s := &Service{
		repo:        repo,
		runtime:     NewTaskRuntime(),
		executor:    executor,
		cronFactory: cronFactory,
	}

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

func (s *Service) All() (map[string]*TaskRecord, error) {
	return s.repo.All()
}

func (s *Service) AddTask(
	cfg *TaskConfig,
) error {

	rec, err := s.repo.Get(cfg.name)
	if err != nil {
		if !errors.Is(err, ErrIsNotExist) {
			return err
		}
	}
	if rec != nil {
		return ErrAlreadyExist
	}

	// validate cron on implementation
	// reglament already checks in Validation but if implemntation
	// is not completely support format it's check this before task has been added
	if _, err := s.cronFactory(cfg.reglament); err != nil {
		return err
	}

	return s.repo.Save(
		&TaskRecord{
			Active:     false,
			TaskConfig: cfg,
		},
	)
}

func (s *Service) Start(id string) error {
	rec, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	cron, err := s.cronFactory(rec.reglament)
	if err != nil {
		return err
	}

	s.runtime.Spawn(rec.TaskConfig, cron, s.executor.execute)
	rec.Active = true

	return s.repo.Save(rec)
}

func (s *Service) Stop(id string) error {

	if _, err := s.repo.Get(id); err != nil {
		return err
	}

	return s.runtime.Kill(id)
}

func (s *Service) Get(id string) (*TaskRecord, error) {
	return s.repo.Get(id)
}

type TaskPatch struct {
	Name        *string     `json:"name,omitempty"`
	Description *string     `json:"description,omitempty"`
	Recipients  *[]agent.ID `json:"recipients,omitempty"`
	Reglament   *string     `json:"schedule,omitempty"`
	Request     *string     `json:"request,omitempty"`
	Oneshot     *bool       `json:"oneshot,omitempty"`
}

func (s *Service) Patch(id string, patch TaskPatch) error {
	record, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	if err := applyPatch(record.TaskConfig, patch); err != nil {
		return err
	}

	return s.repo.Save(record)
}

func applyPatch(cfg *TaskConfig, patch TaskPatch) error {
	if patch.Name != nil {
		cfg.name = *patch.Name
	}

	if patch.Description != nil {
		cfg.description = *patch.Description
	}

	if patch.Recipients != nil {
		cfg.recipients = *patch.Recipients
	}

	if patch.Reglament != nil {
		cfg.reglament = *patch.Reglament
	}

	if patch.Request != nil {
		cfg.request = *patch.Request
	}

	if patch.Oneshot != nil {
		cfg.oneshot = *patch.Oneshot
	}

	return validateTaskConfig(cfg)
}

func (s *Service) Delete(id string) error {
	if err := s.runtime.Kill(id); err != nil {
		return err
	}

	return s.repo.Delete(id)
}

// blocking
func (s *Service) Run(ctx context.Context) {
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

			if err := s.repo.Save(rec); err != nil {
				slog.Error("process done tasks", "task", doneTaskID, "error", err)
			}
		}
	}
}
