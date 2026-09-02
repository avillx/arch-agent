package secrets

import (
	"context"
	"log/slog"
	"maps"
	"sync"
)

type Repo interface {
	Load() (map[string]string, error)
	Save(map[string]string) error
}

type Service struct {
	secrets map[string]string
	repo    Repo
	logger  *slog.Logger

	mu sync.RWMutex
}

func New(repo Repo, logger *slog.Logger) (*Service, error) {

	svc := &Service{
		secrets: map[string]string{},
		repo:    repo,
		logger:  logger.WithGroup("secrets"),
	}

	if err := svc.Reload(context.Background()); err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *Service) Reload(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	secrets, err := s.repo.Load()
	if err != nil {
		return nil
	}

	s.secrets = secrets

	s.logger.Info("reloaded")
	return nil
}

func (s *Service) Get(name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	secret, ok := s.secrets[name]
	return secret, ok
}

func (s *Service) All() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	secrets := maps.Clone(s.secrets)
	return secrets
}

func (s *Service) Put(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.secrets[key] = value

	if err := s.repo.Save(s.secrets); err != nil {
		return err
	}

	s.logger.Info("updated", "secret", key)

	return nil
}

func (s *Service) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.secrets, key)

	if err := s.repo.Save(s.secrets); err != nil {
		return err
	}

	s.logger.Info("deleted", "secret", key)

	return nil
}
