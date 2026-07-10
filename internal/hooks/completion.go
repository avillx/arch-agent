package hooks

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"arch-agent/internal/tools/todo"
	"arch-agent/internal/types"
	"fmt"
	"strings"
)

// empty answer harness
type EmptyAnswerHook struct{}

func (h *EmptyAnswerHook) Apply(c *agent.Completion) (*agent.Completion, error) {
	if c.Content == "" && c.Done {
		c.Done = false
		return c, types.NewAgentMistakeError(prompt.GetEmptyAnswerCautionPrompt())
	}

	return c, nil
}

type todoStorage interface {
	List(session.ID, agent.ID) []todo.TodoItem
}

// todo harness
type undoneTodoHook struct {
	storage todoStorage
	sessID  session.ID
	agentID agent.ID
}

func NewUndoneTodoHook(storage todoStorage, sessID session.ID, agentID agent.ID) *undoneTodoHook {
	return &undoneTodoHook{
		storage: storage,
		sessID:  sessID,
		agentID: agentID,
	}
}

func (h *undoneTodoHook) Apply(c *agent.Completion) (*agent.Completion, error) {

	if !c.Done {
		return c, nil
	}

	var undoneTodos []todo.TodoItem
	for _, item := range h.storage.List(h.sessID, h.agentID) {
		if item.Status != todo.Done {
			undoneTodos = append(undoneTodos, item)
		}
	}

	if len(undoneTodos) > 0 {
		var b strings.Builder
		for _, t := range undoneTodos {
			fmt.Fprintf(&b, "%s\n", t.String())
		}
		c.Done = false

		return c, types.NewAgentMistakeError(prompt.GetUndoneTodosCautionPrompt(b.String()))
	}

	return c, nil
}
