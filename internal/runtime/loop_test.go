package runtime_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/types"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunAgentLoop_ExitsOnDone(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{completion: &agent.Completion{Done: true, Content: "final"}},
		},
	}

	events := runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewUserMessage("hello")},
		nil,
		nil,
	)

	complete, ok := findEvent[*runtime.CompleteEvent](events)
	if !ok {
		t.Fatalf("expected complete event, got %#v", events)
	}
	if complete.Complete().Content != "final" {
		t.Fatalf("unexpected completion content: %q", complete.Complete().Content)
	}

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok {
		t.Fatalf("expected loop exit event, got %#v", events)
	}
	if exit.Err() != nil {
		t.Fatalf("expected clean exit, got %v", exit.Err())
	}
}

func TestRunAgentLoop_ExitsOnMaxTurns(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{"max_turns": 1},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{completion: &agent.Completion{Done: false}},
		},
	}

	events := runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewUserMessage("hello")},
		nil,
		nil,
	)

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok {
		t.Fatalf("expected loop exit event, got %#v", events)
	}
	if exit.Err() == nil || !errors.Is(exit.Err(), runtime.ErrMaxTurnsExceeded) {
		t.Fatalf("expected max turns error, got %v", exit.Err())
	}
}

func TestRunAgentLoop_ExitsOnCancellation(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	events := runLoop(t, ctx, model, []agent.Message{agent.NewUserMessage("hello")}, nil, nil)

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok {
		t.Fatalf("expected loop exit event, got %#v", events)
	}
	if !errors.Is(exit.Err(), context.Canceled) {
		t.Fatalf("expected context canceled, got %v", exit.Err())
	}
}

func TestRunAgentLoop_AbortsOnCompletionError(t *testing.T) {
	errModelExploded := errors.New("model exploded")

	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{err: errModelExploded},
		},
	}

	events := runLoop(t, context.Background(), model, []agent.Message{agent.NewUserMessage("hello")}, nil, nil)

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok {
		t.Fatalf("expected loop exit event, got %#v", events)
	}
	if exit.Err() == nil || !errors.Is(exit.Err(), errModelExploded) {
		t.Fatalf("expected completion error, got %v", exit.Err())
	}
}

func TestRunAgentLoop_ToolCallRoundTrip(t *testing.T) {
	called := false
	tool := &mockTool{
		name: "echo",
		callFn: func(ctx context.Context, args agent.ToolArguments) ([]agent.ContentPart, error) {
			called = true
			return agent.NewContent("tool result"), nil
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
			{completion: &agent.Completion{Done: true, Content: "done"}},
		},
	}

	events := runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewUserMessage("hello")},
		[]agent.Tool{tool},
		nil,
	)

	if !called {
		t.Fatal("expected tool to be called")
	}

	result, ok := findEvent[*runtime.ToolResultEvent](events)
	if !ok {
		t.Fatalf("expected tool result event, got %#v", events)
	}
	if result.Result().Result[0].Text != "tool result" {
		t.Fatalf("unexpected tool result: %#v", result.Result().Result)
	}

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok || exit.Err() != nil {
		t.Fatalf("expected clean exit, got %#v", events)
	}
}

func TestRunAgentLoop_RecoversFromToolPanic(t *testing.T) {
	tool := &mockTool{name: "boom", panic: true}

	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{completion: &agent.Completion{
				Done: false,
				ToolCalls: []*agent.ToolCall{
					agent.NewToolCall("call-1", "boom", agent.ToolArguments(`{}`)),
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
		nil,
	)

	if _, ok := findEvent[*runtime.ToolCallErrEvent](events); !ok {
		t.Fatalf("expected tool call error event, got %#v", events)
	}

	result, ok := findEvent[*runtime.ToolResultEvent](events)
	if !ok {
		t.Fatalf("expected tool result event, got %#v", events)
	}
	if result.Result().Result[0].Text != "toolcall error" {
		t.Fatalf("expected synthesized error result, got %#v", result.Result().Result)
	}

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok || exit.Err() != nil {
		t.Fatalf("expected loop to continue after panic, got %#v", events)
	}
}

func TestRunAgentLoop_ToolAgentMistake(t *testing.T) {
	tool := &mockTool{
		name: "strict",
		callFn: func(ctx context.Context, args agent.ToolArguments) ([]agent.ContentPart, error) {
			return nil, types.NewAgentMistakeError("tool rejected arguments")
		},
	}

	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{completion: &agent.Completion{
				Done: false,
				ToolCalls: []*agent.ToolCall{
					agent.NewToolCall("call-1", "strict", agent.ToolArguments(`{}`)),
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
		nil,
	)

	toolErr, ok := findEvent[*runtime.ToolCallErrEvent](events)
	if !ok {
		t.Fatalf("expected tool call error event, got %#v", events)
	}
	var mistake *types.AgentMistakeError
	if !errors.As(toolErr.Err(), &mistake) {
		t.Fatalf("expected agent mistake, got %v", toolErr.Err())
	}

	result, ok := findEvent[*runtime.ToolResultEvent](events)
	if !ok {
		t.Fatalf("expected tool result event, got %#v", events)
	}
	if !strings.Contains(result.Result().Result[0].Text, "tool rejected arguments") {
		t.Fatalf("expected mistake message in result, got %#v", result.Result().Result)
	}

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok || exit.Err() != nil {
		t.Fatalf("expected loop to continue after tool mistake, got %#v", events)
	}
}

func TestRunAgentLoop_CompletionMistakesExceedLimit(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{completion: &agent.Completion{Done: true, Content: "attempt"}},
			{completion: &agent.Completion{Done: true, Content: "attempt"}},
			{completion: &agent.Completion{Done: true, Content: "attempt"}},
			{completion: &agent.Completion{Done: true, Content: "attempt"}},
		},
	}

	hook := &mockCompletionHook{
		fn: func(ctx context.Context, c *agent.Completion) (*agent.Completion, error) {
			c.Done = false
			return c, types.NewAgentMistakeError("bad completion")
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

	if got := countEvents[*runtime.CompletionMistakeEvent](events); got != 4 {
		t.Fatalf("expected 4 completion mistake events, got %d", got)
	}

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok {
		t.Fatalf("expected loop exit event, got %#v", events)
	}
	if exit.Err() == nil || !errors.Is(exit.Err(), runtime.ErrTooManyMistakes) {
		t.Fatalf("expected mistakes limit error, got %v", exit.Err())
	}
}

func TestRunAgentLoop_CompletionMistakeAddsCaution(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{completion: &agent.Completion{Done: true, Content: "attempt"}},
			{completion: &agent.Completion{Done: true}},
		},
	}

	hookCalls := 0
	hook := &mockCompletionHook{
		fn: func(ctx context.Context, c *agent.Completion) (*agent.Completion, error) {
			hookCalls++
			if hookCalls == 1 {
				c.Done = false
				return c, types.NewAgentMistakeError("model made a mistake")
			}
			return c, nil
		},
	}

	runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewUserMessage("hello")},
		nil,
		[]any{hook},
	)

	calls := model.Calls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 model calls, got %d", len(calls))
	}

	var sawCaution bool
	for _, msg := range calls[1] {
		if msg.Role() != agent.UserMessageRole {
			continue
		}
		if !strings.Contains(msg.String(), "model made a mistake") {
			continue
		}
		sawCaution = true
	}

	if !sawCaution {
		t.Fatalf("expected caution user message in second turn, got %#v", calls[1])
	}
}
