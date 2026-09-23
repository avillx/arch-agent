package runtime_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"context"
	"errors"
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
	errHookFailed := errors.New("hook failed")
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
			return c, errHookFailed
		},
	}

	result, err := runtime.ApplyHooks(
		context.Background(),
		[]any{first, failing},
		completion,
	)
	if !errors.Is(err, errHookFailed) {
		t.Fatalf("expected hook error, got %v", err)
	}
	if result.Content != "second" {
		t.Fatalf("expected last applied value on error, got %q", result.Content)
	}
}
