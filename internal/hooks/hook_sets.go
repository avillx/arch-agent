package hooks

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/logging"
	"path/filepath"
)

const _24kb = 24 * 1024

func NewAgentHooks(
	todoStorage todoStorage,
	r Replcaer,
) ([]any, error) {

	accessRules := []Rule{
		{Pattern: logging.AgentLogFile, Access: Read},
		{Pattern: files.SecretsConfigFile, Access: Read},
		{Pattern: filepath.Join("*", files.AgentFile), Access: No},
		{Pattern: filepath.Join("*", files.SessionsFolder, "**"), Access: No},
		{Pattern: filepath.Join("*", files.ActivityFolder, "**"), Access: Read},
		{Pattern: "**", Access: Write},
	}

	accessHook, err := NewFileAccessHook(accessRules...)
	if err != nil {
		return nil, err
	}

	return []any{
		accessHook,
		// &EmptyAnswerHook{},
		NewUndoneTodoHook(todoStorage),
		&ContentSizeLimitHook{limitBytes: _24kb},
		&NeverReadSecretsHook{r},
		&NeverToolCallSecretsHook{r},
		&NeverTypeSecretsHook{r},
	}, nil
}

func NewMemoryHooksResolver(
	indexer agent.MemoryIndexer,
) (func(agentID agent.ID) []any, error) {

	// for validation path patterns
	_, err := NewMemoryHooks("unexisted_agent", indexer)
	if err != nil {
		return nil, err
	}

	// produce factory
	return func(agentID agent.ID) []any {
		hooks, _ := NewMemoryHooks(agentID, indexer)
		return hooks
	}, nil
}

func NewMemoryHooks(
	agentID agent.ID,
	indexer agent.MemoryIndexer,
) ([]any, error) {

	// Readability helper
	// Concat cwd with pattern and inject agentID in
	// also normalize with filepath
	fp := func(pattern ...string) string {
		sp := filepath.Join(pattern...)
		return filepath.Join(string(agentID), sp)
	}

	accessRules := []Rule{
		{Pattern: fp(files.ActivityFolder, "**"), Access: Read},
		{Pattern: fp(files.MemoryFolder, "**"), Access: Write},
		{Pattern: fp(files.AgentFile), Access: No},
	}

	accessHook, err := NewFileAccessHook(accessRules...)
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
