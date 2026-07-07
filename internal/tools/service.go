package tools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"sync"
)

type ToolServer interface {
	Tools() []agent.Tool
}

type DynamicToolServer interface {
	OnToolsChanged(func() error)
	OnDisconnect(func())
}

type Service struct {
	servers map[string]ToolServer

	mu sync.RWMutex
}

func NewService() *Service {
	return &Service{
		servers: make(map[string]ToolServer),
	}
}

func (s *Service) GetServerTools(servers []string) ([]agent.Tool, error) {

	var errs []error
	tools := []agent.Tool{}
	for _, srv := range servers {
		toolServer, ok := s.servers[srv]
		if ok {
			tools = append(tools, toolServer.Tools()...)
			continue
		}

		errs = append(errs, fmt.Errorf("tool server %s: %w", srv, types.ErrIsNotExist))
	}

	return tools, errors.Join(errs...)
}

func (s *Service) Connect(serverName string, server ToolServer) error {
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
	srv, ok := s.servers[serverName]
	if !ok {
		return fmt.Errorf("server %s not connected", serverName)
	}

	if _, ok := srv.(DynamicToolServer); !ok {
		return fmt.Errorf("server `%s` can't be disconnected", serverName)
	}

	delete(s.servers, serverName)
	return nil
}
