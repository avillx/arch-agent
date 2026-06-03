package tools

import (
	"arch-agent/internal/a2a"
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"context"
	"fmt"
	"log/slog"
	"strings"
)

var _ runtime.PerAgentInstructed = (*CallAgentTool)(nil)

type CallAgentTool struct {
	a2aService *a2a.Service
	agentRepo  agent.Repo
}

func NewCallAgentTool(s *a2a.Service, agentRepo agent.Repo) *CallAgentTool {
	return &CallAgentTool{
		a2aService: s,
		agentRepo:  agentRepo,
	}
}

func (t *CallAgentTool) AgentInstruction(agt agent.Agent) string {
	agents, err := t.agentRepo.All()
	if err != nil {
		slog.Error("call agent tool", "error", "can't gather agents")
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Agent contacts:\n")
	for _, a := range agents {

		if a.ID() == agt.ID() {
			continue
		}

		fmt.Fprintf(&sb, "  * %s - %s\n", string(a.ID()), a.Description())
	}

	return sb.String()
}

func (t *CallAgentTool) Name() agent.ToolName {
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
		Name    agent.ID `json:"name"`
		Request string   `json:"request"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	agentID := MustAgentID(ctx)
	sessionID := MustSessionID(ctx)

	return t.a2aService.Call(
		ctx,
		agentID,
		args.Name,
		sessionID,
		args.Request,
	)
}
