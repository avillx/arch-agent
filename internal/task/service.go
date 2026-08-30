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
	config  TaskConfig
}

type Service struct {
	mu          sync.Mutex
	repo        TaskRepo
	runtimes    map[string]*taskRuntime
	executor    *executor
	cronFactory func(string) (Cron, error)
	agentRepo   agent.Repo
	logger      *slog.Logger
}

func NewService(
	repo TaskRepo,
	executor *executor,
	cronFactory func(string) (Cron, error),
	agentRepo agent.Repo,
	logger *slog.Logger,
) (*Service, error) {

	svc := &Service{
		repo:        repo,
		runtimes:    map[string]*taskRuntime{},
		cronFactory: cronFactory,
		executor:    executor,
		agentRepo:   agentRepo,
		logger:      logger.WithGroup("tasks"),
	}

	tasks, err := repo.All()
	if err != nil {
		return nil, err
	}

	svc.runTasks(tasks)

	return svc, nil
}

// implement sentinel action
func (s *Service) Reload(_ context.Context) error {
	s.logger.Info("reload started")

	tasks, err := s.repo.All()
	if err != nil {
		return err
	}

	loadCandidates := map[string]TaskConfig{}

	s.mu.Lock()
	defer s.mu.Unlock()

	// stop deleted tasks
	for name, rt := range s.runtimes {
		if _, ok := tasks[rt.config.Name]; !ok {
			s.logger.Info("task deleted", "task", name)
			s.stopRuntime(name)
		}
	}

	for name, taskCfg := range tasks {
		rt, ok := s.runtimes[name]
		// new task
		if !ok {
			loadCandidates[name] = taskCfg
			continue
		}

		if taskCfg.Equals(rt.config) {
			continue
		}

		s.stopRuntime(name)
		loadCandidates[name] = taskCfg
	}

	s.runTasks(loadCandidates)

	s.logger.Info("reload finished")

	return nil
}

// run all tasks from config
func (s *Service) runTasks(cfgs map[string]TaskConfig) {
	for _, t := range cfgs {
		if !t.Active {
			continue
		}
		if err := s.start(t); err != nil {
			s.logger.Error("start up", "task", t.Name, "error", err)
		}
	}
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

func (s *Service) start(cfg TaskConfig) error {

	cron, err := s.cronFactory(cfg.Reglament)
	if err != nil {
		return ErrCron
	}

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	rt := &taskRuntime{
		config:  cfg,
		cancel:  cancel,
		stopped: stopped,
	}
	s.runtimes[cfg.Name] = rt

	go func() {
		runLoop(ctx, cron, cfg, s.executor.execute)
		close(stopped)

		s.mu.Lock()
		defer s.mu.Unlock()

		// if task was stoped external (stop invoked not by task itself) then
		// in runtimes, rt pointer not the same.
		// external stopper must:
		// - take lock
		// - eliminate rt pointer from runtimes or shift it to actual
		// TODO: this is a super complex shit. should simplify it.
		if s.runtimes[cfg.Name] != rt {
			return
		}

		s.logger.Info("disabling", "task", cfg.Name)

		// no need to delete runtime pointer from s.runtimes.
		// Because Reload() any way do this.
		cfg.Active = false
		if err := s.repo.Save(cfg); err != nil {
			s.logger.Error("disabling", "task", cfg.Name, "error", err)
		}
	}()

	s.logger.Info("activated", "task", cfg.Name)

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

	return s.repo.Save(cfg)
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

	if next.Equals(cfg) {
		// no need to update
		return nil
	}

	if err := validateTaskConfig(next, s.cronFactory); err != nil {
		return err
	}

	if patch.Name != nil && *patch.Name != cfg.Name {
		if err := s.validateIdentity(*patch.Name); err != nil {
			return err
		}

		// delete depricated task from repo
		if err := s.repo.Delete(cfg.Name); err != nil {
			return err
		}
	}

	return s.repo.Save(next)
}

func (s *Service) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
