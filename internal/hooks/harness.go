package hooks

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"fmt"
)

type cwdBearer interface {
	Cwd() string
}

func NewAgentHarness(b cwdBearer, todoStorage todoStorage) (*runtime.Harness, error) {
	// agent file access
	accessHook, err := NewFileAccessHook(
		b.Cwd(),
		func(agt agent.Agent) []Rule {
			return []Rule{
				{Pattern: ".", Access: No},
				{Pattern: "./shared/*", Access: Write},
				{Pattern: "./skills/*", Access: Read},
				{Pattern: fmt.Sprintf("./%s/*", agt.ID()), Access: Write},
				{Pattern: fmt.Sprintf("./%s/memory/*", agt.ID()), Access: Read},
				{Pattern: fmt.Sprintf("./%s/sessions", agt.ID()), Access: No},
				{Pattern: fmt.Sprintf("./%s/agent.md", agt.ID()), Access: No},
				{Pattern: fmt.Sprintf("./%s/activity/*", agt.ID()), Access: Read},
			}
		},
	)
	if err != nil {
		return nil, err
	}

	undoneTasks := NewUndoneTodoHook(todoStorage)

	return &runtime.Harness{
		OnComplete: runtime.NewHookSet(undoneTasks, &EmptyAnswerHook{}),
		OnToolCall: runtime.NewHookSet(&OnlySupportedExtensionsHook{}, accessHook),
	}, nil
}

func NewMemoryHarness(b cwdBearer, todoStorage todoStorage) *runtime.Harness {

	accessHook, _ := NewFileAccessHook(
		b.Cwd(),
		func(agt agent.Agent) []Rule {
			return []Rule{
				{Pattern: fmt.Sprintf("./%s/*", agt.ID()), Access: Read},
				{Pattern: fmt.Sprintf("./%s/memory/*", agt.ID()), Access: Write},
				{Pattern: fmt.Sprintf("./%s/sessions", agt.ID()), Access: No},
				{Pattern: fmt.Sprintf("./%s/agent.md", agt.ID()), Access: No},
				{Pattern: fmt.Sprintf("./%s/activity/*", agt.ID()), Access: Read},
			}
		},
	)

	undoneTasks := NewUndoneTodoHook(todoStorage)

	return &runtime.Harness{
		OnToolCall: runtime.NewHookSet(accessHook),
		OnComplete: runtime.NewHookSet(undoneTasks, &EmptyAnswerHook{}),
	}
}
