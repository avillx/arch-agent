package runtime_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"context"
	"sync"
	"testing"
	"time"
)

type mockCompletion struct {
	completion *agent.Completion
	err        error
}

type mockModel struct {
	settings   agent.ModelSettings
	ctxLimit   int64
	responses  []mockCompletion
	mu         sync.Mutex
	idx        int
	messages   [][]agent.Message
}

func (m *mockModel) Settings() agent.ModelSettings {
	return m.settings
}

func (m *mockModel) ContextLimit() int64 {
	return m.ctxLimit
}

func (m *mockModel) SupportedModalities() []agent.Modality {
	return []agent.Modality{agent.TextModality}
}

func (m *mockModel) Complete(
	ctx context.Context,
	tools []agent.Tool,
	msgs []agent.Message,
) (*agent.Completion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, msgs)

	if m.idx >= len(m.responses) {
		return &agent.Completion{Done: true}, nil
	}

	resp := m.responses[m.idx]
	m.idx++
	return resp.completion, resp.err
}

func (m *mockModel) Calls() [][]agent.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.messages
}

type mockTool struct {
	name   string
	callFn func(context.Context, agent.ToolArguments) ([]agent.ContentPart, error)
	panic  bool
}

func (t *mockTool) Name() agent.ToolName { return agent.ToolName(t.name) }
func (t *mockTool) Description() string  { return t.name }
func (t *mockTool) Schema() any          { return map[string]any{} }

func (t *mockTool) Call(
	ctx context.Context,
	args agent.ToolArguments,
) ([]agent.ContentPart, error) {
	if t.panic {
		panic("tool boom")
	}
	if t.callFn != nil {
		return t.callFn(ctx, args)
	}
	return agent.NewContent("ok"), nil
}

type mockCompletionHook struct {
	fn func(context.Context, *agent.Completion) (*agent.Completion, error)
}

func (h *mockCompletionHook) Apply(
	ctx context.Context,
	c *agent.Completion,
) (*agent.Completion, error) {
	return h.fn(ctx, c)
}

type mockToolCallHook struct {
	fn func(context.Context, *agent.ToolCall) (*agent.ToolCall, error)
}

func (h *mockToolCallHook) Apply(
	ctx context.Context,
	c *agent.ToolCall,
) (*agent.ToolCall, error) {
	return h.fn(ctx, c)
}

type mockToolResultHook struct {
	fn func(context.Context, *runtime.AfterToolCall) (*runtime.AfterToolCall, error)
}

func (h *mockToolResultHook) Apply(
	ctx context.Context,
	a *runtime.AfterToolCall,
) (*runtime.AfterToolCall, error) {
	return h.fn(ctx, a)
}

func runLoop(
	t *testing.T,
	ctx context.Context,
	model agent.Model,
	msgs []agent.Message,
	tools []agent.Tool,
	hooks []any,
) []runtime.Event {
	t.Helper()

	evCh := make(chan runtime.Event, 64)
	done := make(chan struct{})

	go func() {
		defer close(done)
		runtime.RunAgentLoop(ctx, model, msgs, tools, evCh, hooks)
	}()

	var events []runtime.Event

	for {
		select {
		case ev := <-evCh:
			events = append(events, ev)
			if _, ok := ev.(*runtime.LoopExitEvent); ok {
				<-done
				return events
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("agent loop did not terminate, events so far: %#v", events)
		}
	}
}

func findEvent[T any](events []runtime.Event) (T, bool) {
	for _, ev := range events {
		if typed, ok := ev.(T); ok {
			return typed, true
		}
	}
	var zero T
	return zero, false
}

func countEvents[T any](events []runtime.Event) int {
	n := 0
	for _, ev := range events {
		if _, ok := ev.(T); ok {
			n++
		}
	}
	return n
}