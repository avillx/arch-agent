package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"errors"
)

type Hook[T any] interface {
	Apply(session.ID, agent.Agent, T) (T, error)
}

type HookSet[T any] []Hook[T]

func NewHookSet[T any](hooks ...Hook[T]) HookSet[T] {
	return HookSet[T](hooks)
}

func (s HookSet[T]) Apply(sessID session.ID, agentID agent.Agent, v T) (T, error) {
	var (
		res  = v
		errs = []error{}
	)

	for _, h := range s {
		new, err := h.Apply(sessID, agentID, res)
		if err != nil {
			var agentMistake *types.AgentMistakeError
			if !errors.As(err, &agentMistake) {
				return new, err
			}
			errs = append(errs, err)
			res = new
		}
	}
	return res, errors.Join(errs...)
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
