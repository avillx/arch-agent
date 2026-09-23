package runtime_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestApplyHooks_AppliesInOrder(t *testing.T) {
	completion := &agent.Completion{Content: "start"}

	appendHook := func(suffix string) *mockCompletionHook {
		return &mockCompletionHook{
			fn: func(ctx context.Context, c *agent.Completion) (*agent.Completion, error) {
				c.Content += suffix
				return c, nil
			},
		}
	}

	result, err := runtime.ApplyHooks(
		context.Background(),
		[]any{appendHook("-a"), appendHook("-b")},
		completion,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "start-a-b" {
		t.Fatalf("expected ordered application, got %q", result.Content)
	}
}

func TestApplyHooks_SkipsUnrelatedHooks(t *testing.T) {
	completion := &agent.Completion{Content: "start"}

	unrelated := &mockToolCallHook{
		fn: func(ctx context.Context, c *agent.ToolCall) (*agent.ToolCall, error) {
			t.Fatal("unrelated hook must not be applied")
			return c, nil
		},
	}

	result, err := runtime.ApplyHooks(
		context.Background(),
		[]any{unrelated},
		completion,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "start" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
}

func TestApplyHooks_PropagatesErrorWithCurrentValue(t *testing.T) {
	completion := &agent.Completion{Content: "start"}

	first := &mockCompletionHook{
		fn: func(ctx context.Context, c *agent.Completion) (*agent.Completion, error) {
			c.Content = "first"
			return c, nil
		},
	}
	failing := &mockCompletionHook{
		fn: func(ctx context.Context, c *agent.Completion) (*agent.Completion, error) {
			c.Content = "second"
			return c, errors.New("hook failed")
		},
	}

	result, err := runtime.ApplyHooks(
		context.Background(),
		[]any{first, failing},
		completion,
	)
	if err == nil || err.Error() != "hook failed" {
		t.Fatalf("expected hook error, got %v", err)
	}
	if result.Content != "second" {
		t.Fatalf("expected last applied value on error, got %q", result.Content)
	}
}

func TestRunAgentLoop_AppliesCompletionHook(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{completion: &agent.Completion{Done: true, Content: "raw"}},
		},
	}

	hook := &mockCompletionHook{
		fn: func(ctx context.Context, c *agent.Completion) (*agent.Completion, error) {
			c.Content = "hooked"
			return c, nil
		},
	}

	events := runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewUserMessage("hello")},
		nil,
		[]any{hook},
	)

	complete, ok := findEvent[*runtime.CompleteEvent](events)
	if !ok {
		t.Fatalf("expected complete event, got %#v", events)
	}
	if complete.Complete().Content != "hooked" {
		t.Fatalf("expected hooked completion, got %q", complete.Complete().Content)
	}
}

func TestRunAgentLoop_AppliesToolHooks(t *testing.T) {
	var gotArgs agent.ToolArguments

	tool := &mockTool{
		name: "echo",
		callFn: func(ctx context.Context, args agent.ToolArguments) ([]agent.ContentPart, error) {
			gotArgs = args
			return agent.NewContent("tool result"), nil
		},
	}

	callHook := &mockToolCallHook{
		fn: func(ctx context.Context, c *agent.ToolCall) (*agent.ToolCall, error) {
			c.Arguments = agent.ToolArguments(`{"hooked":true}`)
			return c, nil
		},
	}
	resultHook := &mockToolResultHook{
		fn: func(ctx context.Context, a *runtime.AfterToolCall) (*runtime.AfterToolCall, error) {
			a.ToolResult = agent.NewToolResult(a.ToolCall.ID, "hooked result")
			return a, nil
		},
	}

	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{completion: &agent.Completion{
				Done: false,
				ToolCalls: []*agent.ToolCall{
					agent.NewToolCall("call-1", "echo", agent.ToolArguments(`{}`)),
				},
			}},
			{completion: &agent.Completion{Done: true}},
		},
	}

	events := runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewUserMessage("hello")},
		[]agent.Tool{tool},
		[]any{callHook, resultHook},
	)

	if string(gotArgs) != `{"hooked":true}` {
		t.Fatalf("expected pre-call hook to rewrite args, got %q", string(gotArgs))
	}

	result, ok := findEvent[*runtime.ToolResultEvent](events)
	if !ok {
		t.Fatalf("expected tool result event, got %#v", events)
	}
	if result.Result().Result[0].Text != "hooked result" {
		t.Fatalf("expected post-call hook to rewrite result, got %#v", result.Result().Result)
	}
}

func TestRunAgentLoop_CompletionHookErrorAborts(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{completion: &agent.Completion{Done: true, Content: "raw"}},
		},
	}

	hook := &mockCompletionHook{
		fn: func(ctx context.Context, c *agent.Completion) (*agent.Completion, error) {
			return c, errors.New("hook rejected completion")
		},
	}

	events := runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewUserMessage("hello")},
		nil,
		[]any{hook},
	)

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok {
		t.Fatalf("expected loop exit event, got %#v", events)
	}
	if exit.Err() == nil || !strings.Contains(exit.Err().Error(), "hook rejected completion") {
		t.Fatalf("expected hook error to abort loop, got %v", exit.Err())
	}
}

func TestRunAgentLoop_ToolCallHookErrorIsReported(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{completion: &agent.Completion{
				Done: false,
				ToolCalls: []*agent.ToolCall{
					agent.NewToolCall("call-1", "echo", agent.ToolArguments(`{}`)),
				},
			}},
			{completion: &agent.Completion{Done: true}},
		},
	}

	hook := &mockToolCallHook{
		fn: func(ctx context.Context, c *agent.ToolCall) (*agent.ToolCall, error) {
			return nil, errors.New("hook rejected tool call")
		},
	}

	events := runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewUserMessage("hello")},
		[]agent.Tool{&mockTool{name: "echo"}},
		[]any{hook},
	)

	toolErr, ok := findEvent[*runtime.ToolCallErrEvent](events)
	if !ok {
		t.Fatalf("expected tool call error event, got %#v", events)
	}
	if !strings.Contains(toolErr.Err().Error(), "hook rejected tool call") {
		t.Fatalf("expected hook error in tool call error, got %v", toolErr.Err())
	}

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok || exit.Err() != nil {
		t.Fatalf("expected loop to continue after hook error, got %#v", events)
	}
}