package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	"arch-agent/internal/tools"
	"arch-agent/internal/types"
)

type ConfigRepo interface {
	Load() (map[MCPServerID]ServerGatewayConfig, error)
	Save(MCPServerID, ServerGatewayConfig) error
}

type Service struct {
	toolSvc    *tools.Service
	configRepo ConfigRepo
	servers    map[MCPServerID]MCPServer
	logger     *slog.Logger

	mu sync.RWMutex
}

func NewService(
	ctx context.Context,
	toolSvc *tools.Service,
	repo ConfigRepo,
	logger *slog.Logger,
) (*Service, error) {
	s := &Service{
		toolSvc:    toolSvc,
		configRepo: repo,
		logger:     logger.WithGroup("mcp"),
		servers:    make(map[MCPServerID]MCPServer),
	}

	if err := s.Reload(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Service) Reload(ctx context.Context) error {

	cfgs, err := s.configRepo.Load()
	if err != nil {
		return err
	}

	func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		for _, srv := range s.servers {
			srv.Shutdown()
		}
	}()

	s.loadServers(ctx, cfgs)

	return nil
}

func (s *Service) loadServers(ctx context.Context, cfgs map[MCPServerID]ServerGatewayConfig) {

	var (
		wg sync.WaitGroup
	)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	for id, cfg := range cfgs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Connect(ctx, id, cfg); err != nil {
				// TODO: expose server name
				s.logger.Error("connect server", "error", err)
			}
		}()
	}

	wg.Wait()
}

func (s *Service) List() []MCPServer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Collect(maps.Values(s.servers))
}

func (s *Service) Connect(ctx context.Context, id MCPServerID, cfg ServerGatewayConfig) (MCPServerID, error) {

	srv, err := NewMCPServer(ctx, id, cfg)
	if err != nil {
		return "", fmt.Errorf("mcp: server initialization: %w", err)
	}

	if err := s.ensureServerUnique(srv.ID()); err != nil {
		return "", err
	}

	// if err := s.configRepo.Save(srv.ID(), gatewayToConfig(srv.Gateway())); err != nil {
	// 	return "", err
	// }

	// connect to tool service
	if err := s.toolSvc.Connect(string(srv.ID()), srv); err != nil {
		return "", fmt.Errorf("mcp: register tools: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.servers[srv.ID()] = srv

	logger := s.logger.With("server", srv.ID())

	go func() {
		// blocking
		if err := srv.Run(context.Background()); err != nil {
			logger.Error("bad connection", "error", err)
		}

		if err := s.toolSvc.Disconnect(string(srv.ID())); err != nil {
			logger.Error("bad disconnection", "error", err)
		}

		logger.Info("disconnected")

		if storedSrv, ok := s.servers[srv.ID()]; ok && storedSrv == srv {
			s.mu.Lock()
			defer s.mu.Unlock()

			delete(s.servers, srv.ID())
		}
	}()

	logger.Info("connected")

	return srv.ID(), nil
}

func (s *Service) Disconnect(id MCPServerID) error {

	// s.configRepo.Delete

	s.mu.RLock()
	defer s.mu.RUnlock()

	srv, ok := s.servers[id]
	if !ok {
		return fmt.Errorf("mcp server: %w", types.ErrIsNotExist)
	}

	srv.Shutdown()
	return nil
}

func (s *Service) ensureServerUnique(id MCPServerID) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, srv := range s.servers {
		if srv.ID() == id {
			return fmt.Errorf("mcp: server %s: %w", id, types.ErrAlreadyExist)
		}
	}
	return nil
}

func gatewayToConfig(g gateway) ServerGatewayConfig {

	var cfg ServerGatewayConfig
	switch typed := g.(type) {
	case *httpGateway:
		cfg.HTTPGateway = &HTTPGatewayConfig{
			URL:   typed.url,
			Token: typed.authToken,
		}
	case *processGateway:
		cfg.CommandGateway = &CommandGatewayConfig{
			Command: typed.command,
			Args:    typed.args,
			Env:     typed.env,
		}
	}

	return cfg
}
