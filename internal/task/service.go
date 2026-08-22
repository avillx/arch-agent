package task

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"context"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"
)

type taskRuntime struct {
	cancel  context.CancelFunc
	stopped chan struct{}
}

type Service struct {
	mu          sync.Mutex
	repo        TaskRepo
	runtimes    map[string]*taskRuntime
	executor    *executor
	cronFactory func(string) (Cron, error)
	agentRepo   agent.Repo
}

func NewService(
	repo TaskRepo,
	executor *executor,
	cronFactory func(string) (Cron, error),
	agentRepo agent.Repo,
) (*Service, error) {

	svc := &Service{
		repo:        repo,
		runtimes:    map[string]*taskRuntime{},
		cronFactory: cronFactory,
		executor:    executor,
		agentRepo:   agentRepo,
	}

	tasks, err := repo.All()
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if !t.Active {
			continue
		}
		if err := svc.start(t); err != nil {
			slog.Error("task service startup", "task", t.Name, "error", err)
		}
	}

	return svc, nil
}

func (s *Service) stopRuntime(name string) {
	rt, ok := s.runtimes[name]
	if !ok {
		return
	}
	rt.cancel()
	<-rt.stopped
	delete(s.runtimes, name)
}

func (s *Service) reconcileTask(cfg TaskConfig) error {
	s.stopRuntime(cfg.Name)

	if err := s.repo.Save(cfg); err != nil {
		return err
	}

	if cfg.Active {
		return s.start(cfg)
	}

	return nil
}

func (s *Service) start(cfg TaskConfig) error {

	cron, err := s.cronFactory(cfg.Reglament)
	if err != nil {
		return ErrCron
	}

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	rt := &taskRuntime{cancel: cancel, stopped: stopped}
	s.runtimes[cfg.Name] = rt

	go func() {
		runLoop(ctx, cron, cfg, s.executor.execute)
		close(stopped)

		s.mu.Lock()
		defer s.mu.Unlock()

		if s.runtimes[cfg.Name] != rt {
			return
		}
		delete(s.runtimes, cfg.Name)
		cfg.Active = false
		if err := s.repo.Save(cfg); err != nil {
			slog.Error("persist disabled task", "task", cfg.Name, "error", err)
		}
	}()
	return nil
}

func runLoop(ctx context.Context, cron Cron, cfg TaskConfig, onExecute func(context.Context, TaskConfig)) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(cron.NextTime()):
			onExecute(ctx, cfg)
			if cfg.Oneshot {
				return
			}
		}
	}
}

func (s *Service) List() ([]TaskConfig, error) {

	taskMap, err := s.repo.All()
	if err != nil {
		return nil, err
	}

	taskConfigs := slices.Collect(maps.Values(taskMap))
	if taskConfigs == nil {
		taskConfigs = []TaskConfig{}
	}

	return taskConfigs, nil
}

func (s *Service) Add(cfg TaskConfig) error {

	if err := validateTaskConfig(cfg, s.cronFactory); err != nil {
		return err
	}

	if err := s.validateRecipients(cfg.Recipients); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.validateIdentity(cfg.Name); err != nil {
		return err
	}

	return s.reconcileTask(cfg)
}

func (s *Service) Patch(name string, patch TaskPatch) error {

	if patch.Recipients != nil {
		if err := s.validateRecipients(*patch.Recipients); err != nil {
			return err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.repo.Get(name)
	if err != nil {
		return err
	}

	next := applyPatch(cfg, patch)

	if err := validateTaskConfig(next, s.cronFactory); err != nil {
		return err
	}

	var errs []error
	if patch.Name != nil && *patch.Name != cfg.Name {
		if err := s.validateIdentity(*patch.Name); err != nil {
			return err
		}

		s.stopRuntime(cfg.Name)
		if err := s.repo.Delete(cfg.Name); err != nil {
			errs = append(errs, err)
		}
	}

	if err := s.reconcileTask(next); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (s *Service) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopRuntime(name)

	return s.repo.Delete(name)
}

func (s *Service) validateRecipients(recipients []agent.ID) error {
	problems := map[string]string{}
	for _, r := range recipients {
		if _, err := s.agentRepo.Get(r); err != nil {
			if errors.Is(err, types.ErrIsNotExist) {
				problems[string(r)] = "is not exist"
				continue
			}
			return err
		}
	}

	if len(problems) > 0 {
		return types.NewValidationError(problems)
	}

	return nil
}

func (s *Service) validateIdentity(id string) error {
	_, err := s.repo.Get(id)
	if err != nil {
		if errors.Is(err, ErrIsNotExist) {
			return nil
		}
		return err
	}
	return types.NewValidationError(map[string]string{id: "already exist"})
}
