package hooks

import (
	"arch-agent/internal/agent"
	"fmt"
)

const _24kb = 24 * 1024

type cwdBearer interface {
	Cwd() string
}

func NewAgentHooks(
	b cwdBearer,
	todoStorage todoStorage,
) ([]any, error) {

	accessRules := []Rule{
		{Pattern: ".", Access: Write},
		{Pattern: "./*/activity/*", Access: Read},
		{Pattern: "./*/memory/*", Access: Read},
		{Pattern: "./*/sessions", Access: No},
		{Pattern: "./*/agent.md", Access: No},
	}

	accessHook, err := NewFileAccessHook(b.Cwd(), accessRules...)
	if err != nil {
		return nil, err
	}

	return []any{
		accessHook,
		&EmptyAnswerHook{},
		&UndoneTodoHook{storage: todoStorage},
		&OnlySupportedExtensionsHook{},
		&ContentSizeLimitHook{limitBytes: _24kb},
	}, nil
}

func NewMemoryHooksResolver(
	b cwdBearer,
	indexer agent.MemoryIndexer,
) (func(agentID agent.ID) []any, error) {

	// for validation path patterns
	_, err := NewMemoryHooks("unexisted_agent", b, indexer)
	if err != nil {
		return nil, err
	}

	// produce factory
	return func(agentID agent.ID) []any {
		hooks, _ := NewMemoryHooks(agentID, b, indexer)
		return hooks
	}, nil
}

func NewMemoryHooks(
	agentID agent.ID,
	b cwdBearer,
	indexer agent.MemoryIndexer,
) ([]any, error) {

	accessRules := []Rule{
		{Pattern: ".", Access: No},
		{Pattern: fmt.Sprintf("./%s/*", agentID), Access: Read},
		{Pattern: fmt.Sprintf("./%s/sessions", agentID), Access: No},
		{Pattern: fmt.Sprintf("./%s/agent.md", agentID), Access: No},
		{Pattern: fmt.Sprintf("./%s/memory/*", agentID), Access: Write},
		{Pattern: fmt.Sprintf("./%s/activity/*", agentID), Access: Read},
	}

	accessHook, err := NewFileAccessHook(b.Cwd(), accessRules...)
	if err != nil {
		return nil, err
	}

	return []any{
		accessHook,
		// &UndoneTodoHook{storage: todoStorage},
		&OnlyValidMemoryFrontmatterHook{indexer: indexer},
		&EmptyAnswerHook{},
	}, nil
}
