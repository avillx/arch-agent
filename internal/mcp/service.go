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
)

type ConfigRepo interface {
	Load() (map[MCPServerID]ServerGatewayConfig, error)
	Save(MCPServerID, ServerGatewayConfig) error
	Delete(MCPServerID) error
}

type Service struct {
	toolSvc    *tools.Service
	configRepo ConfigRepo
	logger     *slog.Logger

	servers map[MCPServerID]MCPServer
	mu      sync.RWMutex
}

func NewService(
	ctx context.Context,
	toolSvc *tools.Service,
	repo ConfigRepo,
	logger *slog.Logger,
) (*Service, error) {
	svc := &Service{
		toolSvc:    toolSvc,
		configRepo: repo,
		logger:     logger.WithGroup("mcp"),
		servers:    make(map[MCPServerID]MCPServer),
	}

	if err := svc.load(ctx); err != nil {
		return nil, err
	}

	return svc, nil
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
				delete(s.servers, id)
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
				delete(s.servers, id)
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

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	var wg sync.WaitGroup

	for id, cfg := range cfgs {
		wg.Go(func() {
			if err := s.connectServer(ctx, id, cfg); err != nil {
				s.logger.Error("connect server", "server", id, "error", err)
			}
		})
	}

	wg.Wait()
}

func (s *Service) connectServer(ctx context.Context, id MCPServerID, cfg ServerGatewayConfig) error {

	srv, err := NewMCPServer(ctx, id, cfg)
	if err != nil {
		return fmt.Errorf("mcp: server initialization: %w", err)
	}

	// connect to tool service
	if err := s.toolSvc.Connect(string(srv.ID()), srv); err != nil {
		return fmt.Errorf("mcp: register tools: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.servers[srv.ID()] = srv

	go func() {
		logger := s.logger.With("server", srv.ID())
		// blocking
		if err := srv.Run(context.Background()); err != nil {
			logger.Error("bad connection", "error", err)
			srv.setErr(err)
		}

		s.toolSvc.Disconnect(string(srv.ID()))

		logger.Info("disconnected")
	}()

	s.logger.Info("connected", "server", srv.ID())

	return nil
}

func (s *Service) DeleteServer(id MCPServerID) error {
	return s.configRepo.Delete(id)
}

// Override behaviour
func (s *Service) SetServer(ctx context.Context, id MCPServerID, cfg ServerGatewayConfig) error {

	var depricatedSrv MCPServer
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		if srv, ok := s.servers[id]; ok {
			depricatedSrv = srv
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// NOTE: mcp service has detect of already connected servers
	// and not trying to override succecceful working servers
	// thaths the reason to connect it here directly.
	// Cause if something going wrong func return error and not
	// trying to swallow it on reload attempt. Reload leaves
	// this connection untouched
	if err := s.connectServer(ctx, id, cfg); err != nil {
		return err
	}

	// NOTE: if something goes worng on edit server config then it
	// never set on s.servers a new bad server
	// connectServer can't return error after server is setted. if behaviour
	// has been changed then this solutuin should changed too
	// To prvent gorutine leak it shutdown's here
	if depricatedSrv != nil {
		depricatedSrv.Shutdown()
	}

	return s.configRepo.Save(id, cfg)
}
