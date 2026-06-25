package tools

import (
	"arch-agent/internal/agent"
	"fmt"
	"sync"
)

type ToolServer interface {
	Name() string
	Tools() []agent.Tool
	OnToolsChanged(func() error)
	OnDisconnect(func())
}

type Service struct {
	mu      sync.RWMutex
	tools   map[agent.ToolName]agent.Tool
	owned   map[string][]agent.ToolName
	servers map[string]ToolServer
}

func NewService() *Service {
	s := &Service{
		tools:   make(map[agent.ToolName]agent.Tool),
		owned:   make(map[string][]agent.ToolName),
		servers: make(map[string]ToolServer),
	}
	return s
}

func (s *Service) AddTools(tools ...agent.Tool) error {
	return s.register("", tools)
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

func (s *Service) GetTools(toolNames []agent.ToolName) ([]agent.Tool, error) {

	tools := make([]agent.Tool, len(toolNames))
	for i, name := range toolNames {
		t, ok := s.tools[name]
		if !ok {
			return nil, fmt.Errorf("tool %s is not exist", name)
		}
		tools[i] = t
	}

	return tools, nil
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

	server.OnToolsChanged(func() error {
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

	server.OnDisconnect(func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.unregister(name)
		delete(s.servers, name)
	})

	return nil
}

func (s *Service) Disconnect(serverName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.servers[serverName]; !ok {
		return fmt.Errorf("server %s not connected", serverName)
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
