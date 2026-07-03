package task

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"context"
	"errors"
	"fmt"
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
	agentRepo   agent.Repo
	repo        TaskRepo
}

func NewService(
	ctx context.Context,
	repo TaskRepo,
	agentRepo agent.Repo,
	cronFactory func(string) (Cron, error),
	executor *executor,
) (*Service, error) {

	s := &Service{
		repo:        repo,
		agentRepo:   agentRepo,
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

var ErrNoRecipients = errors.New("task must contain at least one recipient")

func (s *Service) AddTask(
	cfg *TaskConfig,
) error {

	if err := s.validateOnTaskUniqness(cfg.name); err != nil {
		return err
	}

	if err := s.validateRecipients(cfg.recipients); err != nil {
		return err
	}

	if err := s.validateOnCronExpression(cfg.reglament); err != nil {
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

	rec.Active = true

	onStop := func() error {
		rec.Active = false
		return s.repo.Save(rec)
	}

	runningTask, err := newRunningTask(rec.TaskConfig, cron, s.executor.execute, onStop)
	if err != nil {
		return err
	}

	if err := s.runtime.Start(runningTask); err != nil {
		return err
	}

	if err := s.repo.Save(rec); err != nil {
		runningTask.stop()
		return err
	}

	return nil
}

func (s *Service) Stop(id string) error {

	// check existence
	if _, err := s.repo.Get(id); err != nil {
		return err
	}

	return s.runtime.Kill(id)
}

func (s *Service) Get(id string) (*TaskRecord, error) {
	return s.repo.Get(id)
}

func (s *Service) Delete(id string) error {

	// check existence
	if _, err := s.repo.Get(id); err != nil {
		return err
	}

	if err := s.runtime.Kill(id); err != nil {
		if !errors.Is(err, ErrTaskIsNotRunning) {
			return err
		}
	}

	return s.repo.Delete(id)
}

// blocking
// func (s *Service) Run(ctx context.Context) {
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case doneTaskID := <-s.runtime.DoneChannel():
// 			rec, err := s.repo.Get(doneTaskID)
// 			if err != nil {
// 				slog.Error("process done tasks", "task", doneTaskID, "error", err)
// 				return
// 			}

// 			rec.Active = false // panic here rec is nil after deletion

// 			if err := s.repo.Save(rec); err != nil {
// 				slog.Error("process done tasks", "task", doneTaskID, "error", err)
// 			}
// 		}
// 	}
// }

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

	if patch.Name != nil {
		if err := s.validateOnTaskUniqness(*patch.Name); err != nil {
			return err
		}
	}

	if patch.Recipients != nil {
		if err := s.validateRecipients(*patch.Recipients); err != nil {
			return err
		}
	}

	if patch.Reglament != nil {
		if err := s.validateOnCronExpression(*patch.Reglament); err != nil {
			return err
		}
	}

	if err := applyPatch(record.TaskConfig, patch); err != nil {
		return err
	}

	if err := s.repo.Save(record); err != nil {
		return err
	}

	if patch.Name != nil && id != *patch.Name {
		return s.Delete(id)
	}

	return nil
}

func (s *Service) validateRecipients(recipients []agent.ID) error {
	if !(len(recipients) > 0) {
		return ErrNoRecipients
	}

	for _, r := range recipients {
		if _, err := s.agentRepo.Get(r); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) validateOnTaskUniqness(id string) error {
	exist, err := s.repo.Get(id)
	if err != nil && !errors.Is(err, ErrIsNotExist) {
		return err
	}
	if exist != nil {
		return fmt.Errorf("task %s: %w", id, ErrAlreadyExist)
	}
	return nil
}

func (s *Service) validateOnCronExpression(exp string) error {
	if _, err := s.cronFactory(exp); err != nil {
		return ErrCron
	}
	return nil
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
