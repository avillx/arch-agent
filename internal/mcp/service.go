package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"arch-agent/internal/tools"
	"arch-agent/internal/types"
)

const (
	healthInterval = 30 * time.Second
	initTimeout    = 15 * time.Second
)

type ConfigRepo interface {
	Load() ([]ServerConfig, error)
	Save(ServerConfig) error
}

type Service struct {
	toolSvc    *tools.Service
	configRepo ConfigRepo
	servers    map[MCPServerID]*MCPServer
	mu         sync.RWMutex
}

func NewService(ctx context.Context, toolSvc *tools.Service, repo ConfigRepo) (*Service, error) {
	s := &Service{
		toolSvc:    toolSvc,
		configRepo: repo,
		servers:    make(map[MCPServerID]*MCPServer),
	}

	if err := s.loadServers(ctx); err != nil {
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

func (s *Service) loadServers(ctx context.Context) error {
	cfgs, err := s.configRepo.Load()
	if err != nil {
		return err
	}
	var errs []error
	for _, cfg := range cfgs {

		gateway, err := createGateway(cfg.ServerGatewayConfig)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		srv, err := NewMCPServer(gateway, WithState(cfg.ID, cfg.Connected))
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// no identity check.
		// if repo return the dublicated cfg's then problem is in repo
		s.servers[srv.ID] = srv

		if cfg.Connected {
			if err := s.Connect(ctx, cfg.ID); err != nil {
				errs = append(errs, err)
				continue
			}
		}
	}

	return errors.Join(errs...)
}

func (s *Service) List() []*MCPServer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Collect(maps.Values(s.servers))
}

type ServerConfig struct {
	ID                  MCPServerID `json:"id"`
	Connected           bool        `json:"connected"`
	ServerGatewayConfig `json:"gateway"`
}

type ServerGatewayConfig struct {
	HTTP   *GatewayHTTPConfig `json:"http,omitempty"`
	NPM    *GatewayNPMConfig  `json:"npm,omitempty"`
	Binary *GatewayBinConfig  `json:"bin,omitempty"`
}

type GatewayHTTPConfig struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`
}

type GatewayNPMConfig struct {
	Package string            `json:"package"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type GatewayBinConfig struct {
	Path string            `json:"path,omitempty"`
	Args []string          `json:"args,omitempty"`
	Env  map[string]string `json:"env,omitempty"`
}

func validateConfig(cfg ServerGatewayConfig) error {

	problems := map[string]string{}

	// TODO: add more checks
	// e.g. ->
	// map[string]string{
	// 	"config": "config must contain only one gateway",
	// },

	if cfg.HTTP == nil &&
		cfg.NPM == nil &&
		cfg.Binary == nil {

		problems["config"] = "must be at least one gateway (http,npm,bin)"
	}

	if len(problems) > 0 {
		return types.NewValidationError(problems)
	}

	return nil
}

func createGateway(cfg ServerGatewayConfig) (gateway, error) {
	switch {
	case cfg.HTTP != nil:
		return newHTTPGateway(cfg.HTTP.URL, cfg.HTTP.Token), nil
	case cfg.NPM != nil:
		return newNPMProcessGateway(cfg.NPM.Package, cfg.NPM.Args, cfg.NPM.Env)
	case cfg.Binary != nil:
		return newBinaryProcessGateway(cfg.Binary.Path, cfg.Binary.Args, cfg.Binary.Env)
	default:
		return nil, fmt.Errorf("mcp add has no gateway")
	}
}

func (s *Service) Add(ctx context.Context, cfg ServerGatewayConfig) (MCPServerID, error) {

	if err := validateConfig(cfg); err != nil {
		return "", err
	}

	gateway, err := createGateway(cfg)
	if err != nil {
		return "", err
	}

	srv, err := NewMCPServer(gateway, WithInit(ctx))
	if err != nil {
		return "", fmt.Errorf("mcp: server initialization: %w", err)
	}

	if err := s.ensureServerUnique(srv.ID); err != nil {
		return "", err
	}

	if err := s.saveServer(srv); err != nil {
		return "", err
	}

	if err := s.toolSvc.Connect(srv); err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.servers[srv.ID] = srv

	return srv.ID, nil
}

func (s *Service) Remove(id MCPServerID) error {
	if err := s.Disconnect(id); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.servers, id)

	return nil
}

func (s *Service) Connect(ctx context.Context, id MCPServerID) error {

	// connect to mcp server
	ctx, cancel := context.WithTimeout(ctx, initTimeout)
	defer cancel()

	srv, err := s.getServer(id)
	if err != nil {
		return err
	}

	if err := srv.Connect(ctx); err != nil {
		return err
	}

	srv.OnDisconnect(func() {
		s.saveServer(srv)
	})

	// connect to tool service
	if err := s.toolSvc.Connect(srv); err != nil {
		srv.Disconnect()
		return fmt.Errorf("mcp: register tools: %w", err)
	}

	return s.saveServer(srv)
}

func (s *Service) Disconnect(id MCPServerID) error {
	srv, err := s.getServer(id)
	if err != nil {
		return err
	}

	srv.Disconnect()
	return nil
}

// safe obtain server ptr
func (s *Service) getServer(id MCPServerID) (*MCPServer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	srv, ok := s.servers[id]
	if !ok {
		return nil, fmt.Errorf("mcp: server %s not found", id)
	}

	return srv, nil
}

// safe put server ptr
func (s *Service) saveServer(srv *MCPServer) error {
	return s.configRepo.Save(ServerConfig{
		ID:                  srv.ID,
		Connected:           srv.Connected,
		ServerGatewayConfig: gatewayToConfig(srv.gateway),
	})
}

func gatewayToConfig(g gateway) ServerGatewayConfig {

	var cfg ServerGatewayConfig
	switch typed := g.(type) {
	case *httpGateway:
		cfg.HTTP = &GatewayHTTPConfig{
			URL:   typed.url,
			Token: typed.authToken,
		}
	case *processGateway:
		if strings.HasSuffix(typed.command, "npx") {
			cfg.NPM = &GatewayNPMConfig{
				Package: typed.command,
				Args:    typed.args,
				Env:     typed.env,
			}
		} else {
			cfg.Binary = &GatewayBinConfig{
				Path: typed.command,
				Args: typed.args,
				Env:  typed.env,
			}
		}
	}

	return cfg
}

// validation on url server uniqueness
func (s *Service) ensureServerUnique(id MCPServerID) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, srv := range s.servers {
		if srv.ID == id {
			return fmt.Errorf("mcp: server %s: %w", id, types.ErrAlreadyExist)
		}
	}
	return nil
}
