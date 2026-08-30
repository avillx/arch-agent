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

	if err := s.load(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Service) List() []MCPServer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Collect(maps.Values(s.servers))
}

func (s *Service) Reload(ctx context.Context) error {
	s.logger.Info("reload started")
	cfgs, err := s.configRepo.Load()
	if err != nil {
		return err
	}

	loadCandidates := map[MCPServerID]ServerGatewayConfig{}

	func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		// shutdown deleted servers
		for id, srv := range s.servers {
			if _, ok := cfgs[id]; !ok {
				srv.Shutdown()
			}
		}

		// gather load candidates
		for id, cfg := range cfgs {

			srv, ok := s.servers[id]
			// new added servers
			if !ok {
				loadCandidates[id] = cfg
				continue
			}

			// if config has updated
			if !cfg.Equals(srv.Config()) {
				srv.Shutdown()
				loadCandidates[id] = cfg
			}
		}
	}()

	s.connectServers(ctx, loadCandidates)
	s.logger.Info("reload finished")

	return nil
}

func (s *Service) load(ctx context.Context) error {
	cfgs, err := s.configRepo.Load()
	if err != nil {
		return err
	}
	s.connectServers(ctx, cfgs)
	return nil
}

func (s *Service) connectServers(ctx context.Context, cfgs map[MCPServerID]ServerGatewayConfig) {

	var (
		wg sync.WaitGroup
	)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	for id, cfg := range cfgs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.connectServer(ctx, id, cfg); err != nil {
				s.logger.Error("connect server", "server", id, "error", err)
			}
		}()
	}

	wg.Wait()
}

func (s *Service) connectServer(ctx context.Context, id MCPServerID, cfg ServerGatewayConfig) (MCPServerID, error) {

	srv, err := NewMCPServer(ctx, id, cfg)
	if err != nil {
		return "", fmt.Errorf("mcp: server initialization: %w", err)
	}

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

// TODO: funcs for api
// AddServer (add server if repo | sentinel trigger reload automaticly)
// DeleteServer (disconnect and delete server from config |)
