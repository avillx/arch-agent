package executioncontext_test

import (
	executioncontext "arch-agent/internal/app/executioncontext"
	tools "arch-agent/internal/app/toolexecutor"
	"arch-agent/internal/domain/conversation"
	"context"
	"slices"
	"testing"
)

// helpers

func makeTool(name string, reasonOnce bool) tools.ToolDefinition {
	return tools.ToolDefinition{Name: name, ReasonOnce: reasonOnce}
}

func makeToolCall(toolName string) *conversation.ToolCall {
	return conversation.NewToolCall("id-"+toolName, toolName, nil)
}

func newCtx(toolDefs []tools.ToolDefinition) *executioncontext.ExecutionContext {
	return executioncontext.NewExecutionContext(
		executioncontext.Reflection{},
		executioncontext.Memory{},
		executioncontext.AgentConfig{},
		toolDefs,
	)
}

func toolNames(params executioncontext.ReasonParams) []string {
	names := make([]string, len(params.Tools))
	for i, t := range params.Tools {
		names[i] = t.Name
	}
	return names
}

// --- ShouldFollowUp ---

func TestShouldFollowUp_NoCalls_False(t *testing.T) {
	ec := newCtx([]tools.ToolDefinition{makeTool("tool-a", false)})

	if ec.ShouldFollowUp(nil) {
		t.Error("expected false for empty calls")
	}
}

func TestShouldFollowUp_ReasonOnceTool_False(t *testing.T) {
	ec := newCtx([]tools.ToolDefinition{makeTool("tool-a", true)})
	calls := []*conversation.ToolCall{makeToolCall("tool-a")}

	if ec.ShouldFollowUp(calls) {
		t.Error("expected false for ReasonOnce tool")
	}
}

func TestShouldFollowUp_FollowUpRequiredTool_True(t *testing.T) {
	ec := newCtx([]tools.ToolDefinition{makeTool("tool-a", false)})
	calls := []*conversation.ToolCall{makeToolCall("tool-a")}

	if !ec.ShouldFollowUp(calls) {
		t.Error("expected true for follow up required tool with budget")
	}
}

func TestShouldFollowUp_UnknownTool_False(t *testing.T) {
	ec := newCtx([]tools.ToolDefinition{makeTool("tool-a", false)})
	calls := []*conversation.ToolCall{makeToolCall("unknown")}

	if ec.ShouldFollowUp(calls) {
		t.Error("expected false for unknown tool")
	}
}

// --- NextReasonParams + бюджет ---

func TestNextReasonParams_DecrementsBudget(t *testing.T) {
	ec := newCtx([]tools.ToolDefinition{makeTool("tool-a", false)})
	calls := []*conversation.ToolCall{makeToolCall("tool-a")}

	// после 9 вызовов бюджет = 1, followUp ещё должен работать
	for range executioncontext.DefaultFollowUpBudget - 1 {
		ec.NextReasonParams(context.Background(), nil)
	}

	if !ec.ShouldFollowUp(calls) {
		t.Error("expected followUp to still work before budget exhausted")
	}
}

func TestNextReasonParams_BudgetExhausted_ShouldFollowUpFalse(t *testing.T) {
	ec := newCtx([]tools.ToolDefinition{makeTool("tool-a", false)})
	calls := []*conversation.ToolCall{makeToolCall("tool-a")}

	// 10 вызовов — бюджет обнуляется
	for range executioncontext.DefaultFollowUpBudget {
		ec.NextReasonParams(context.Background(), nil)
	}

	if ec.ShouldFollowUp(calls) {
		t.Error("expected false after budget exhausted")
	}
}

// --- excludeFollowUpRequiredTools мутация ---

func TestNextReasonParams_OnBudgetZero_FollowUpRequiredToolsExcluded(t *testing.T) {
	ec := newCtx([]tools.ToolDefinition{
		makeTool("follow up required tool", false),
		makeTool("once-tool", true),
	})

	for range executioncontext.DefaultFollowUpBudget {
		ec.NextReasonParams(context.Background(), nil)
	}

	// после исчерпания бюджета запрашиваем ещё раз — follow up required tool должен исчезнуть
	params := ec.NextReasonParams(context.Background(), nil)
	names := toolNames(params)

	if slices.Contains(names, "follow up required tool") {
		t.Error("follow up required tool should be excluded after budget exhausted")
	}
	if !slices.Contains(names, "once-tool") {
		t.Error("once-tool should remain after budget exhausted")
	}
}

func TestNextReasonParams_BeforeBudgetZero_AllToolsPresent(t *testing.T) {
	ec := newCtx([]tools.ToolDefinition{
		makeTool("follow up required tool", false),
		makeTool("once-tool", true),
	})

	params := ec.NextReasonParams(context.Background(), nil)
	names := toolNames(params)

	if !slices.Contains(names, "follow up required tool") {
		t.Error("follow up required tool should be present before budget exhausted")
	}
	if !slices.Contains(names, "once-tool") {
		t.Error("once-tool should be present before budget exhausted")
	}
}
