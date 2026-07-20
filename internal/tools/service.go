package tools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"sync"
)

type Service struct {
	servers map[string]agent.ToolServer

	mu sync.RWMutex
}

func NewService() *Service {
	return &Service{
		servers: make(map[string]agent.ToolServer),
	}
}

func (s *Service) AllToolServers(names ...string) map[string]agent.ToolServer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.servers
}

func (s *Service) ToolServers(names ...string) ([]agent.ToolServer, error) {

	s.mu.RLock()
	defer s.mu.RUnlock()

	// validate exisntence
	errs := []error{}
	servers := []agent.ToolServer{}
	for _, n := range names {
		if srv, ok := s.servers[n]; ok {
			servers = append(servers, srv)
			continue
		}
		errs = append(errs, fmt.Errorf("tool service: %s : %w", n, types.ErrIsNotExist))
	}

	return servers, errors.Join(errs...)
}

func (s *Service) Connect(serverName string, server agent.ToolServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.servers[serverName]; exists {
		return fmt.Errorf("server %s already connected", serverName)
	}
	s.servers[serverName] = server

	return nil
}

func (s *Service) Disconnect(serverName string) error {

	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.servers[serverName]
	if !ok {
		return fmt.Errorf("server %s not connected", serverName)
	}

	delete(s.servers, serverName)
	return nil
}
