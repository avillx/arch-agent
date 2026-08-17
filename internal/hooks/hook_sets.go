package hooks

import (
	"arch-agent/internal/agent"
	"path/filepath"
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
		{Pattern: filepath.Join(b.Cwd(), "*/agent.md"), Access: No},
		{Pattern: filepath.Join(b.Cwd(), "*/sessions/**"), Access: No},
		{Pattern: filepath.Join(b.Cwd(), "*/activity/**"), Access: Read},
		{Pattern: filepath.Join(b.Cwd(), "*/memory/**"), Access: Read},
		{Pattern: filepath.Join(b.Cwd(), "**"), Access: Write},
	}

	accessHook, err := NewFileAccessHook(b.Cwd(), accessRules...)
	if err != nil {
		return nil, err
	}

	return []any{
		accessHook,
		&EmptyAnswerHook{},
		NewUndoneTodoHook(todoStorage),
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

	// Readability helper
	// Concat cwd with pattern and inject agentID in
	// also normalize with filepath
	cwdAndAgentFolder := func(pattern string) string {
		return filepath.Join(b.Cwd(), string(agentID), pattern)
	}

	accessRules := []Rule{
		{Pattern: cwdAndAgentFolder("activity/**"), Access: Read},
		{Pattern: cwdAndAgentFolder("memory/**"), Access: Write},
		{Pattern: cwdAndAgentFolder("agent.md"), Access: No},
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
