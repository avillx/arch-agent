package tools

import (
	"arch-agent/internal/a2a"
	"arch-agent/internal/agent"
	"context"
	"fmt"
	"strings"
)

// getAgents
type GetAgentsTool struct {
	a2aService *a2a.Service
}

func NewGetAgentsTool(s *a2a.Service) *GetAgentsTool {
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

func (t *GetAgentsTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{}
}

func (t *GetAgentsTool) Call(ctx context.Context, _ agent.ToolArguments) (string, error) {

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
	a2aService *a2a.Service
}

func NewCallAgentTool(s *a2a.Service) *CallAgentTool {
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

func (t *CallAgentTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "name",
			Required:    true,
			Type:        agent.TypeString,
			Description: "name of agent",
		},
		{
			Name:        "request",
			Required:    true,
			Type:        agent.TypeString,
			Description: "This is your message to another agent",
		},
	}
}

func (t *CallAgentTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
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
