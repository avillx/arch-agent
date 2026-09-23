package runtime_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
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

	msgs := []agent.Message{agent.NewUserMessage("hello")}
	events := runLoop(t, context.Background(), model, msgs, nil, nil)

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

	if len(result.Result().Result) <= 0 {
		t.Fatalf("expected synthesized result, got nil result")
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
	if len(result.Result().Result) <= 0 {
		t.Fatalf("expected synthesized result, got nil result")
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
		// NOTE: well this important part of a logic but only way to ensure this shit
		// is a string comparation. So is a smell yeah
		if !strings.Contains(msg.String(), "model made a mistake") {
			continue
		}
		sawCaution = true
	}

	if !sawCaution {
		t.Fatalf("expected caution user message in second turn, got %#v", calls[1])
	}
}

func TestRunAgentLoop_CompactsOnOverflow(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{err: runtime.ErrContextOverflow},
			{completion: &agent.Completion{Content: "compacted summary"}},
			{completion: &agent.Completion{Done: true, Content: "final"}},
		},
	}

	msgs := []agent.Message{
		agent.NewSystemMessage("system"),
		agent.NewUserMessage("turn 1"),
		agent.NewUserMessage("turn 2"),
		agent.NewUserMessage("turn 3"),
	}

	events := runLoop(t, context.Background(), model, msgs, nil, nil)

	compaction, ok := findEvent[*runtime.CompactionEvent](events)
	if !ok {
		t.Fatalf("expected compaction event, got %#v", events)
	}
	if compaction.Summary() != "compacted summary" {
		t.Fatalf("unexpected summary: %q", compaction.Summary())
	}

	compacted := compaction.CompactedContext()
	if len(compacted) == 0 {
		t.Fatal("expected non-empty compacted context")
	}
	if system, ok := compacted[0].(*agent.SystemMessage); !ok || system.Content()[0].Text != "system" {
		t.Fatalf("expected system message to be preserved first, got %#v", compacted[0])
	}

	calls := model.Calls()
	if len(calls) < 2 {
		t.Fatalf("expected at least 2 model calls, got %d", len(calls))
	}

	compactionCall := calls[1]
	if len(compactionCall) == 0 {
		t.Fatal("expected compaction call to carry messages")
	}
	system, ok := compactionCall[0].(*agent.SystemMessage)
	if !ok || system.Content()[0].Text != prompt.Compaction() {
		t.Fatalf("expected compaction prompt as system message, got %#v", compactionCall[0])
	}

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok || exit.Err() != nil {
		t.Fatalf("expected clean exit after compaction, got %#v", events)
	}
}

func TestRunAgentLoop_CompactsOnThreshold(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100,
		responses: []mockCompletion{
			{completion: &agent.Completion{
				Done:             false,
				InputTokens:      95,
				CompletionTokens: 10,
			}},
			{completion: &agent.Completion{Content: "compacted summary"}},
			{completion: &agent.Completion{Done: true, Content: "final"}},
		},
	}

	events := runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewSystemMessage("system"), agent.NewUserMessage("hello")},
		nil,
		nil,
	)

	compaction, ok := findEvent[*runtime.CompactionEvent](events)
	if !ok {
		t.Fatalf("expected compaction event, got %#v", events)
	}
	if compaction.Summary() != "compacted summary" {
		t.Fatalf("unexpected summary: %q", compaction.Summary())
	}

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok || exit.Err() != nil {
		t.Fatalf("expected clean exit, got %#v", events)
	}
}

func TestRunAgentLoop_CompactionFailureExits(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{err: runtime.ErrContextOverflow},
			{err: context.DeadlineExceeded},
		},
	}

	events := runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewSystemMessage("system"), agent.NewUserMessage("hello")},
		nil,
		nil,
	)

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok {
		t.Fatalf("expected loop exit event, got %#v", events)
	}
	if exit.Err() == nil || !errors.Is(exit.Err(), context.DeadlineExceeded) {
		t.Fatalf("expected unhandled overflow error, got %v", exit.Err())
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
	errHookRejectedCompletion := errors.New("hook rejected completion")

	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{completion: &agent.Completion{Done: true, Content: "raw"}},
		},
	}

	hook := &mockCompletionHook{
		fn: func(ctx context.Context, c *agent.Completion) (*agent.Completion, error) {
			return c, errHookRejectedCompletion
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
	if exit.Err() == nil || !errors.Is(exit.Err(), errHookRejectedCompletion) {
		t.Fatalf("expected hook error to abort loop, got %v", exit.Err())
	}
}

func TestRunAgentLoop_ToolCallHookErrorIsReported(t *testing.T) {
	errHookRejectedToolCall := errors.New("hook rejected tool call")

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
			return nil, errHookRejectedToolCall
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
	if !errors.Is(toolErr.Err(), errHookRejectedToolCall) {
		t.Fatalf("expected hook error in tool call error, got %v", toolErr.Err())
	}

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok || exit.Err() != nil {
		t.Fatalf("expected loop to continue after hook error, got %#v", events)
	}
}
