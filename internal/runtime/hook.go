package runtime

import (
	"arch-agent/internal/agent"
	"context"
)

type CompletionHook Hook[*agent.Completion]
type ToolCallHook Hook[*agent.ToolCall]
type ToolResultHook Hook[AfterToolCall]

type Hook[T any] interface {
	Apply(context.Context, T) (T, error)
}

type AfterToolCall struct {
	*agent.ToolCall
	*agent.ToolResult
}

func ApplyHooks[T any](ctx context.Context, hooks []any, event T) (T, error) {

	for _, h := range hooks {

		typedHook, ok := h.(Hook[T])
		if !ok {
			continue
		}

		new, err := typedHook.Apply(ctx, event)
		if err != nil {
			return new, err
		}
		event = new

	}

	return event, nil
}
