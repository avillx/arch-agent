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
	"log/slog"
	"strings"
	"sync"
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

const maxIncomplitedTodoAttempts = 3

// todo harness
var _ runtime.CompletionHook = (*UndoneTodoHook)(nil)

type UndoneTodoHook struct {
	storage todoStorage

	attemptCounter map[session.ID]int
	mu             sync.Mutex
}

func NewUndoneTodoHook(storage todoStorage) *UndoneTodoHook {
	return &UndoneTodoHook{
		storage:        storage,
		attemptCounter: map[session.ID]int{},
	}
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
		return c, fmt.Errorf("todo hook: has no agent ID")
	}
	sessID, ok := chat.SessionIDFromContext(ctx)
	if !ok {
		return c, fmt.Errorf("todo hook: has no session ID")
	}

	undoneTodos := h.extractUndoneTodos(sessID, agentID)

	if len(undoneTodos) > 0 {

		taskList := todoToList(undoneTodos)

		if h.isAgentCantHandle(sessID) {

			h.dropAttempts(sessID)
			slog.Warn(
				"undone todo hook",
				"cause", "agent handle with undone tasks",
				"agent", agentID,
				"session", sessID,
				"tasks", taskList,
			)
			return c, nil
		}

		h.increaseAttempts(sessID)

		c.Done = false

		caution := prompt.UndoneTodosCaution(taskList)
		return c, types.NewAgentMistakeError(caution)
	}

	h.dropAttempts(sessID)

	return c, nil
}

func (h *UndoneTodoHook) extractUndoneTodos(sessID session.ID, agentID agent.ID) []todo.TodoItem {
	var undoneTodos []todo.TodoItem
	for _, item := range h.storage.List(sessID, agentID) {
		if item.Status != todo.Done && item.Status != todo.Declined {
			undoneTodos = append(undoneTodos, item)
		}
	}
	return undoneTodos
}

func (h *UndoneTodoHook) increaseAttempts(sessID session.ID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.attemptCounter[sessID]++
}

func (h *UndoneTodoHook) dropAttempts(sessID session.ID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.attemptCounter, sessID)
}

func (h *UndoneTodoHook) isAgentCantHandle(sessID session.ID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.attemptCounter[sessID] >= maxIncomplitedTodoAttempts {
		return true

	}
	return false
}

func todoToList(todoItems []todo.TodoItem) string {
	var b strings.Builder
	for _, t := range todoItems {
		fmt.Fprintf(&b, "%s\n", t.String())
	}
	return b.String()
}
