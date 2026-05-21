package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/tool"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

type ToolServer interface {
	Name() string
	Tools() []tool.Tool
	OnToolsChanged(func() error)
	OnDisconnect(func())
}

type ToolService struct {
	mu      sync.RWMutex
	tools   map[string]tool.Tool
	owned   map[string][]string
	servers map[string]ToolServer
}

func NewToolService() *ToolService {
	s := &ToolService{
		tools:   make(map[string]tool.Tool),
		owned:   make(map[string][]string),
		servers: make(map[string]ToolServer),
	}
	return s
}

func (s *ToolService) AddTools(tools ...tool.Tool) error {
	return s.register("", tools)
}

func (s *ToolService) Tools() []tool.Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]tool.Tool, 0, len(s.tools))
	for _, t := range s.tools {
		result = append(result, t)
	}
	return result
}

func (s *ToolService) GetTools(toolNames []string) ([]tool.Tool, error) {

	tools := make([]tool.Tool, len(toolNames))
	for i, name := range toolNames {
		t, ok := s.tools[name]
		if !ok {
			return nil, fmt.Errorf("tool %s is not exist", name)
		}
		tools[i] = t
	}

	return tools, nil
}

func (s *ToolService) Connect(server ToolServer) error {
	name := server.Name()

	s.mu.Lock()
	if _, exists := s.servers[name]; exists {
		s.mu.Unlock()
		return fmt.Errorf("server %s already connected", name)
	}
	if err := s.register(name, server.Tools()); err != nil {
		s.mu.Unlock()
		return err
	}
	s.servers[name] = server
	s.mu.Unlock()

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

func (s *ToolService) Disconnect(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.servers[name]; !ok {
		return fmt.Errorf("server %s not connected", name)
	}
	s.unregister(name)
	delete(s.servers, name)
	return nil
}

func (s *ToolService) register(owner string, tools []tool.Tool) error {
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

func (s *ToolService) unregister(owner string) {
	for _, name := range s.owned[owner] {
		delete(s.tools, name)
	}
	delete(s.owned, owner)
}

func UnwrapArgs[T any](raw tool.ToolArguments) (T, error) {
	var args T
	if err := json.Unmarshal(raw, &args); err != nil {
		return args, err
	}
	return args, nil
}

func MustAgentID(ctx context.Context) agent.ID {
	agentID, ok := agent.IDFromContext(ctx)
	if !ok {
		slog.Error("Critical context has no agentID")
	}
	return agentID
}
