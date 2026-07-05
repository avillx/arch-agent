package tools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"sync"
)

type ToolServer interface {
	Name() string
	Tools() []agent.Tool
}

type DynamicToolServer interface {
	OnToolsChanged(func() error)
	OnDisconnect(func())
}

type Service struct {
	mu      sync.RWMutex
	tools   map[agent.ToolName]agent.Tool
	owned   map[string][]agent.ToolName
	servers map[string]ToolServer
}

func NewService(servers ...ToolServer) (*Service, error) {
	s := &Service{
		tools:   make(map[agent.ToolName]agent.Tool),
		owned:   make(map[string][]agent.ToolName),
		servers: make(map[string]ToolServer),
	}

	for _, srv := range servers {
		if err := s.Connect(srv); err != nil {
			return nil, fmt.Errorf("can't connect to build in tool server %s", srv.Name())
		}
	}

	return s, nil
}

func (s *Service) Tools() []agent.Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]agent.Tool, 0, len(s.tools))
	for _, t := range s.tools {
		result = append(result, t)
	}
	return result
}

func (s *Service) GetToolsByServers(servers []string) ([]agent.Tool, error) {

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

func (s *Service) GetTools(toolNames []agent.ToolName) ([]agent.Tool, error) {

	var errs []error
	tools := make([]agent.Tool, 0, len(toolNames))
	for _, name := range toolNames {
		t, ok := s.tools[name]
		if !ok {
			errs = append(errs, fmt.Errorf("tool %s: %w", name, types.ErrIsNotExist))
		}
		tools = append(tools, t)
	}

	return tools, errors.Join(errs...)
}

func (s *Service) Connect(server ToolServer) error {
	name := server.Name()

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.servers[name]; exists {
		return fmt.Errorf("server %s already connected", name)
	}
	if err := s.register(name, server.Tools()); err != nil {
		return err
	}
	s.servers[name] = server

	if dynamicTS, ok := server.(DynamicToolServer); ok {
		dynamicTS.OnToolsChanged(func() error {
			s.mu.Lock()
			defer s.mu.Unlock()

			if _, ok := s.servers[name]; !ok {
				return nil
			}
			s.unregister(name)
			if err := s.register(name, server.Tools()); err != nil {
				delete(s.servers, name)
				return err
			}
			return nil
		})

		dynamicTS.OnDisconnect(func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			s.unregister(name)
			delete(s.servers, name)
		})
	}

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
		return fmt.Errorf("server `%s` can't be disconnected", srv.Name())
	}

	s.unregister(serverName)
	delete(s.servers, serverName)
	return nil
}

func (s *Service) register(owner string, tools []agent.Tool) error {
	for _, t := range tools {
		if _, exists := s.tools[t.Name()]; exists {
			return fmt.Errorf("tool %q already registered", t.Name())
		}
	}
	for _, t := range tools {
		s.tools[t.Name()] = t
		s.owned[owner] = append(s.owned[owner], t.Name())
	}
	return nil
}

func (s *Service) unregister(owner string) {
	for _, name := range s.owned[owner] {
		delete(s.tools, name)
	}
	delete(s.owned, owner)
}
