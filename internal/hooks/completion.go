package hooks

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/prompt"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"arch-agent/internal/tools/todo"
	"arch-agent/internal/types"
	"context"
	"fmt"
	"strings"
)

// ensure valid memory front matter
var _ runtime.CompletionHook = (*OnlyValidMemoryFrontmatterHook)(nil)

type OnlyValidMemoryFrontmatterHook struct {
	agentID agent.ID
	indexer agent.MemoryIndexer
}

func (h *OnlyValidMemoryFrontmatterHook) Apply(
	ctx context.Context,
	c *agent.Completion,
) (*agent.Completion, error) {

	if !c.Done {
		return c, nil
	}

	if _, err := h.indexer.MemoryIndex(h.agentID); err != nil {
		if joinedErrs, ok := err.(interface{ Unwrap() []error }); ok {
			var sb strings.Builder
			for _, e := range joinedErrs.Unwrap() {
				sb.WriteString(e.Error())
			}
			c.Done = false
			return c, types.NewAgentMistakeError(sb.String())
		}
	}

	return c, nil
}

// empty answer harness
var _ runtime.CompletionHook = (*EmptyAnswerHook)(nil)

type EmptyAnswerHook struct{}

func (h *EmptyAnswerHook) Apply(
	ctx context.Context,
	c *agent.Completion,
) (*agent.Completion, error) {
	if c.Content == "" && c.Done {
		c.Done = false
		return c, types.NewAgentMistakeError(prompt.EmptyAnswerCaution())
	}

	return c, nil
}

type todoStorage interface {
	List(session.ID, agent.ID) []todo.TodoItem
}

// todo harness
var _ runtime.CompletionHook = (*UndoneTodoHook)(nil)

type UndoneTodoHook struct {
	storage todoStorage
}

func (h *UndoneTodoHook) Apply(
	ctx context.Context,
	c *agent.Completion,
) (*agent.Completion, error) {

	if !c.Done {
		return c, nil
	}

	agentID, ok := chat.AgentIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("todo hook: has no agent ID")
	}
	sessID, ok := chat.SessionIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("todo hook: has no session ID")
	}

	var undoneTodos []todo.TodoItem
	for _, item := range h.storage.List(sessID, agentID) {
		if item.Status != todo.Done && item.Status != todo.Declined {
			undoneTodos = append(undoneTodos, item)
		}
	}

	if len(undoneTodos) > 0 {
		var b strings.Builder
		for _, t := range undoneTodos {
			fmt.Fprintf(&b, "%s\n", t.String())
		}
		c.Done = false

		return c, types.NewAgentMistakeError(prompt.UndoneTodosCaution(b.String()))
	}

	return c, nil
}
