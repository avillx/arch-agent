package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
)

type Hook[T any] interface {
	Apply(session.ID, agent.Agent, T) (T, error)
}

type HookSet[T any] []Hook[T]

func NewHookSet[T any](hooks ...Hook[T]) HookSet[T] {
	return HookSet[T](hooks)
}

func (s HookSet[T]) Apply(sessID session.ID, agentID agent.Agent, v T) (T, error) {
	for _, h := range s {
		new, err := h.Apply(sessID, agentID, v)
		if err != nil {
			return new, err
		}
		v = new
	}
	return v, nil
}

type AfterToolCall struct {
	*agent.ToolCall
	*agent.ToolResult
}

type Harness struct {
	OnToolCall              HookSet[*agent.ToolCall]
	OnToolCallResultMessage HookSet[*AfterToolCall]
	OnComplete              HookSet[*agent.Completion]
}
