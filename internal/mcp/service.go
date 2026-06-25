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

const (
	healthInterval = 30 * time.Second
	initTimeout    = 15 * time.Second
)

type Service struct {
	toolSvc *tools.Service
	repo    MCPRepo
	servers map[MCPServerID]*MCPServer
	mu      sync.RWMutex
}

func NewService(ctx context.Context, toolSvc *tools.Service, repo MCPRepo) (*Service, error) {
	stored, err := repo.Load()
	if err != nil {
		return nil, fmt.Errorf("mcp service: load repo: %w", err)
	}

	s := &Service{
		toolSvc: toolSvc,
		repo:    repo,
		servers: make(map[MCPServerID]*MCPServer),
	}

	for _, srv := range stored {
		s.servers[srv.ID] = srv
		if srv.Connected {
			if err := s.Connect(ctx, srv.ID); err != nil {
				slog.Error("mcp service, servers connection", "error", err)
				srv.Connected = false
			}
		}
	}

	return s, s.flush() // flush to commit servers bad connection attempts
}

func (s *Service) List() []*MCPServer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Collect(maps.Values(s.servers))
}

func (s *Service) Add(ctx context.Context, url string) (MCPServerID, error) {

	if err := s.ensureServerUnique(url); err != nil {
		return "", err
	}

	// ctx, cancel := context.WithTimeout(context.Background(), initTimeout)
	// defer cancel()

	srv, err := NewMCPServer(WithInit(ctx, url))
	if err != nil {
		return "", fmt.Errorf("mcp: server initialization: %w", err)
	}

	if err := s.putServer(srv); err != nil {
		return "", err
	}

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

	if err := tryConnect(ctx, srv, DefaultMaxAttempts); err != nil {
		return err
	}

	srv.OnDisconnect(func() {
		s.flush()
	})

	// connect to tool service
	if err := s.toolSvc.Connect(srv); err != nil {
		srv.Disconnect()
		return fmt.Errorf("mcp: register tools: %w", err)
	}

	return s.flush()
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
func (s *Service) putServer(srv *MCPServer) error {

	// check existence
	_, err := s.getServer(srv.ID)
	if err == nil {
		return fmt.Errorf("server %s already exists", srv.ID)
	}

	s.mu.RLock()
	s.servers[srv.ID] = srv
	s.mu.RUnlock()

	return s.flush()
}

// validation on url server uniqueness
func (s *Service) ensureServerUnique(url string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, srv := range s.servers {
		if srv.URL == url {
			return fmt.Errorf("mcp: server with url %s already added", url)
		}
	}
	return nil
}

// save current state to repo
func (s *Service) flush() error {
	return s.repo.Save(s.List())
}
