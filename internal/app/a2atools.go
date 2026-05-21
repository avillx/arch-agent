package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/tool"
	"context"
	"fmt"
	"strings"
)

// getAgents
type GetAgentsTool struct {
	a2aService *A2AService
}

func NewGetAgentsTool(s *A2AService) *GetAgentsTool {
	return &GetAgentsTool{
		a2aService: s,
	}
}

func (t *GetAgentsTool) Name() string {
	return "get_agents"
}

func (t *GetAgentsTool) Description() string {
	return "return a list of available agents"
}

func (t *GetAgentsTool) Schema() []tool.ToolProperty {
	return []tool.ToolProperty{}
}

func (t *GetAgentsTool) Call(ctx context.Context, _ tool.ToolArguments) (string, error) {

	agentID := MustAgentID(ctx)

	contacts, err := t.a2aService.AgentContacts(agent.ID(agentID))
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("## Agents")
	for _, contact := range contacts {
		sb.WriteString(fmt.Sprintf("* %s - %s\n", contact.ID, contact.CallGuide))
	}

	return sb.String(), nil
}

// callAgentTool
type CallAgentTool struct {
	a2aService *A2AService
}

func NewCallAgentTool(s *A2AService) *CallAgentTool {
	return &CallAgentTool{
		a2aService: s,
	}
}

func (t *CallAgentTool) Name() string {
	return "call_agent"
}

func (t *CallAgentTool) Description() string {
	return "send request to another agent"
}

func (t *CallAgentTool) Schema() []tool.ToolProperty {
	return []tool.ToolProperty{
		{
			Name:        "name",
			Required:    true,
			Type:        tool.TypeString,
			Description: "name of agent",
		},
		{
			Name:        "request",
			Required:    true,
			Type:        tool.TypeString,
			Description: "This is your message to another agent",
		},
	}
}

func (t *CallAgentTool) Call(ctx context.Context, rawArgs tool.ToolArguments) (string, error) {
	args, err := UnwrapArgs[struct {
		Name    string `json:"name"`
		Request string `json:"request"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	agentID := MustAgentID(ctx)

	contacts, err := t.a2aService.AgentContacts(agent.ID(agentID))
	if err != nil {
		return "", err
	}

	for _, contact := range contacts {
		if contact.ID == agent.ID(args.Name) {
			return t.a2aService.Call(
				context.Background(),
				agent.ID(agentID),
				agent.ID(contact.ID),
				args.Request,
			)
		}
	}

	return "", fmt.Errorf("agent %s not found", args.Name)
}
