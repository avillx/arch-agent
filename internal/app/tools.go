package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/types"
	"context"
	"log/slog"
	"maps"
	"slices"
)

type Server interface {
	Name() string
	ToolGuide(agent.ID) string
	Tools() []types.ToolDefinition
	SendCall(ctx context.Context, call *types.ToolCall, sign agent.ID) (string, error)
}

type ToolService struct {
	servers map[string]Server
}

func NewToolService() *ToolService {
	return &ToolService{
		servers: map[string]Server{},
	}
}

func (s *ToolService) Servers() []Server { return slices.Collect(maps.Values(s.servers)) }

func (s *ToolService) Disconnect(serverID string) {
	delete(s.servers, serverID)
}

func (s *ToolService) ToolKit(id agent.ID, serverNames []string) *AgentToolKit {
	servers := []Server{}
	for _, serverName := range serverNames {
		if server, ok := s.servers[serverName]; ok {
			servers = append(servers, server)
			continue
		}
		slog.Error("server not exist %s", "error", serverName)
	}
	return NewAgentToolKit(id, servers...)
}

func (s *ToolService) Connect(server Server) {
	s.servers[server.Name()] = server
}
