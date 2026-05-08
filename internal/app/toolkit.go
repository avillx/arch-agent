package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/types"
	"context"
	"errors"
	"fmt"
	"strings"
)

type AgentToolKit struct {
	agentID agent.ID
	routes  map[string]ToolServer
}

func NewAgentToolKit(id agent.ID, servers ...ToolServer) *AgentToolKit {
	routes := map[string]ToolServer{}

	for _, server := range servers {
		for _, serverTool := range server.Tools() {
			routes[serverTool.Name] = server
		}
	}

	return &AgentToolKit{
		agentID: id,
		routes:  routes,
	}
}

func (s *AgentToolKit) ToolGuides() string {
	var instructions strings.Builder
	for _, server := range s.routes {
		instructions.WriteString(server.ToolGuide(s.agentID) + "\n\n")
	}
	return instructions.String()
}

func (s *AgentToolKit) Tools() []types.ToolDefinition {
	toolDefs := []types.ToolDefinition{}
	for _, server := range s.routes {
		toolDefs = append(toolDefs, server.Tools()...)
	}
	return toolDefs
}

func (s *AgentToolKit) SendCall(ctx context.Context, call *types.ToolCall) types.Message {

	// get server by toolname
	server, ok := s.routes[call.ToolName]
	if !ok {
		content := fmt.Sprintf("tool %s is not exist", call.ToolName)
		return types.NewToolResultMessage(call.ID, content)
	}

	// send call
	result, err := server.SendCall(ctx, call, s.agentID)
	if err != nil {
		result += errors.Join(err, types.ErrBadToolCall).Error()
	}

	return types.NewToolResultMessage(call.ID, result)
}
