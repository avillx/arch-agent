package hooks

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"fmt"
)

type cwdBearer interface {
	Cwd() string
}

func ProduceHarnessFactory(b cwdBearer, todoStorage todoStorage) func(sessionID session.ID, agentID agent.ID) *runtime.Harness {
	return func(sessionID session.ID, agentID agent.ID) *runtime.Harness {

		// agent file access
		accessHook, _ := NewFileAccessHook(
			b.Cwd(),
			Rule{Pattern: ".", Access: No},
			Rule{Pattern: "./shared/*", Access: Write},
			Rule{Pattern: "./skills/*", Access: Read},
			Rule{Pattern: fmt.Sprintf("./%s/*", agentID), Access: Write},
			Rule{Pattern: fmt.Sprintf("./%s/memory/*", agentID), Access: Read},
			Rule{Pattern: fmt.Sprintf("./%s/sessions", agentID), Access: No},
			Rule{Pattern: fmt.Sprintf("./%s/agent.md", agentID), Access: No},
			Rule{Pattern: fmt.Sprintf("./%s/activity/*", agentID), Access: Read},
		)

		completionHook := NewUndoneTodoHook(todoStorage, sessionID, agentID)

		return &runtime.Harness{
			OnToolCall: runtime.NewHookSet(accessHook),
			OnComplete: runtime.NewHookSet(completionHook, &EmptyAnswerHook{}),
		}
	}
}
