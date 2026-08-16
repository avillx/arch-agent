package runtime

import (
	"arch-agent/internal/agent"
)

type Hook[T any] interface {
	Apply(T) (T, error)
}

type AfterToolCall struct {
	*agent.ToolCall
	*agent.ToolResult
}

type Harness struct {
	OnToolCall              []Hook[*agent.ToolCall]
	OnToolCallResultMessage []Hook[*AfterToolCall]
	OnComplete              []Hook[*agent.Completion]
}

func ApplyHooks[T any](hooks []any, event T) (T, error) {

	for _, h := range hooks {

		typedHook, ok := h.(Hook[T])
		if !ok {
			continue
		}

		new, err := typedHook.Apply(event)
		if err != nil {
			return new, err
		}
		event = new

	}

	return event, nil
}
