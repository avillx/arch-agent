package tools

import (
	"arch-agent/internal/a2a"
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
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
	const instruction = `## Call Agents:
You can delegate task to other agents with diffirent capabilities.
Also can call another yourself instance for keep context clean,
do it when operation too complex (5+ toolcalls).
Request should be exhaustive: e.g. task, details, context and expected result.
Called agent is stateless. If agent clarify something
then agent need full request with clarificaton again.
`

	agents, err := t.agentRepo.All()
	if err != nil {
		slog.Error("call agent tool", "error", "can't gather agents")
		return ""
	}

	var sb strings.Builder

	sb.WriteString(instruction + "\n")
	sb.WriteString("Agent contacts:\n")
	for _, a := range agents {

		isYouLabel := ""
		if a.ID() == agt.ID() {
			isYouLabel = "(You)"
		}

		fmt.Fprintf(&sb, "* %s%s - %s\n", string(a.ID()), isYouLabel, a.Description())
	}

	return sb.String()
}

func (t *CallAgentTool) Name() agent.ToolName {
	return "call_agent"
}

func (t *CallAgentTool) Description() string {
	return "Delegate a task or question to another agent"
}

func (t *CallAgentTool) TimeOut() time.Duration {
	return 10 * time.Minute
}

func (t *CallAgentTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:        "name",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Target agent name",
		},
		{
			Name:        "request",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Task or question to send to the agent",
		},
	}
}

func (t *CallAgentTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := UnwrapArgs[struct {
		Name    agent.ID `json:"name"`
		Request string   `json:"request"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	agentID := MustAgentID(ctx)
	sessionID := MustSessionID(ctx)

	res, err := t.a2aService.Call(
		ctx,
		agentID,
		args.Name,
		sessionID,
		args.Request,
	)

	if err != nil {
		res = fmt.Sprintf("%s. agent %s has errors when processing your request", res, args.Name)
	} else {
		res = fmt.Sprintf("# Agent %s respones:\n%s", args.Name, res)
	}

	return Result(res), err
}
