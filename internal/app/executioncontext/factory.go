package executioncontext

import (
	tools "arch-agent/internal/app/toolexecutor"
	"arch-agent/internal/domain/conversation"
	"context"
)

type MemoryProvider interface {
	Snapshot(ctx context.Context, conversation []conversation.Message) Memory
}

type RequestContextFactory struct {
	memoryProvider MemoryProvider
	reflector      Reflector
}

func NewRequestContextFactory(p MemoryProvider, reflector Reflector) *RequestContextFactory {
	return &RequestContextFactory{
		memoryProvider: p,
		reflector:      reflector,
	}
}

func (f *RequestContextFactory) Build(ctx context.Context, a AgentConfig, messages []conversation.Message, tools []tools.ToolDefinition) (*ExecutionContext, error) {

	memory := f.memoryProvider.Snapshot(ctx, messages)
	reflection := f.reflector.Reflect(ctx, messages, a.Personality)

	return NewExecutionContext(reflection, memory, a, tools), nil
}

// func AssymblyMemory(m *Memory) string {
// 	var builder strings.Builder

// 	for _, s := range m.Semantic {
// 		builder.WriteString(s.String())
// 		builder.WriteString("\n")
// 	}

// 	for _, e := range m.sortedEpisodes() {
// 		builder.WriteString(e.String())
// 		builder.WriteString("\n")
// 	}

// 	return builder.String()
// }
