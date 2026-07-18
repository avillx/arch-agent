package mcp

import (
	"context"
	"errors"
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
	Load() ([]ServerGatewayConfig, error)
	Save(MCPServerID, ServerGatewayConfig) error
}

type Service struct {
	toolSvc    *tools.Service
	configRepo ConfigRepo
	servers    map[MCPServerID]MCPServer
	mu         sync.RWMutex
}

func NewService(ctx context.Context, toolSvc *tools.Service, repo ConfigRepo) (*Service, error) {
	s := &Service{
		toolSvc:    toolSvc,
		configRepo: repo,
		servers:    make(map[MCPServerID]MCPServer),
	}

	cfgs, err := s.configRepo.Load()
	if err != nil {
		return nil, fmt.Errorf("mcp repo: %w", err)
	}

	if err := s.loadServers(ctx, cfgs); err != nil {
		if wraped, ok := err.(interface{ Unwrap() []error }); ok {
			for _, e := range wraped.Unwrap() {
				slog.Error("mcp service: server init: config", "error", e)
			}
		} else {
			slog.Error("mcp service: server init", "error", err)
		}
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

	return s.loadServers(ctx, cfgs)
}

func (s *Service) loadServers(ctx context.Context, cfgs []ServerGatewayConfig) error {

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs = []error{}
	)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	for _, cfg := range cfgs {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if _, err := s.Connect(ctx, cfg); err != nil {
				mu.Lock()
				defer mu.Unlock()
				errs = append(errs, err)
			}
		}()
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (s *Service) List() []MCPServer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Collect(maps.Values(s.servers))
}

func (s *Service) Connect(ctx context.Context, cfg ServerGatewayConfig) (MCPServerID, error) {
	srv, err := NewMCPServer(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("mcp: server initialization: %w", err)
	}

	if err := s.ensureServerUnique(srv.ID()); err != nil {
		return "", err
	}

	if err := s.configRepo.Save(srv.ID(), gatewayToConfig(srv.Gateway())); err != nil {
		return "", err
	}

	// connect to tool service
	if err := s.toolSvc.Connect(string(srv.ID()), srv); err != nil {
		return "", fmt.Errorf("mcp: register tools: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.servers[srv.ID()] = srv

	go func() {
		// blocking
		if err := srv.Run(context.Background()); err != nil {
			slog.Error("mcp server connection", "error", err)
		}
		slog.Info("mcp server disconnected", "server", srv.ID, "error", err)

		if terr := s.toolSvc.Disconnect(string(srv.ID())); terr != nil {
			slog.Error("mcp disconnection", "error", terr)
		}

		if storedSrv, ok := s.servers[srv.ID()]; ok && storedSrv == srv {
			s.mu.Lock()
			defer s.mu.Unlock()

			delete(s.servers, srv.ID())
		}
	}()

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
