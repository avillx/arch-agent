package executioncontext

import (
	tools "arch-agent/internal/app/toolexecutor"
	"arch-agent/internal/domain/conversation"
	"context"
	"maps"
	"slices"
)

const DefaultFollowUpBudget = 10

type ReasonParams struct {
	Agent              AgentConfig
	Reflection         *Reflection
	ContextDescription string
	Memory             Memory
	Messages           []conversation.Message
	Tools              []tools.ToolDefinition
}

type ExecutionContext struct {
	agent              AgentConfig
	ContextDescription string
	reflection         *Reflection
	memory             Memory
	followUpBudget     int
	tools              map[string]tools.ToolDefinition
}

func NewExecutionContext(
	reflection *Reflection,
	ContextDescription string,
	memory Memory,
	agent AgentConfig,
	tools []tools.ToolDefinition,
) *ExecutionContext {
	return &ExecutionContext{
		agent:          agent,
		reflection:     reflection,
		memory:         memory,
		followUpBudget: DefaultFollowUpBudget,
		tools:          toolSliceToMap(tools),
	}
}

func (r *ExecutionContext) NextReasonParams(ctx context.Context, messasges []conversation.Message) ReasonParams {

	if r.followUpBudget--; r.followUpBudget == 0 {
		r.excludeFollowUpRequiredTools()
	}

	return ReasonParams{
		Agent:              r.agent,
		Reflection:         r.reflection,
		Messages:           messasges,
		Memory:             r.memory,
		ContextDescription: r.ContextDescription,
		// map to slice
		Tools: slices.Collect(maps.Values(r.tools)),
	}
}

func (r *ExecutionContext) ShouldFollowUp(calls []*conversation.ToolCall) bool {

	followUpByToolCall := slices.ContainsFunc(calls, func(c *conversation.ToolCall) bool {
		t, ok := r.tools[c.ToolName()]
		return ok && !t.ReasonOnce
	})

	if r.followUpBudget > 0 && followUpByToolCall {
		return true
	}

	return false
}

func (r *ExecutionContext) excludeFollowUpRequiredTools() {
	maps.DeleteFunc(r.tools, func(_ string, t tools.ToolDefinition) bool {
		return !t.ReasonOnce
	})
}

func toolSliceToMap(toolDefs []tools.ToolDefinition) map[string]tools.ToolDefinition {
	var m = map[string]tools.ToolDefinition{}
	for _, t := range toolDefs {
		m[t.Name] = t
	}

	return m
}
