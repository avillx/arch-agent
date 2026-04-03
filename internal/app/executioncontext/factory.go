package executioncontext

import (
	"arch-agent/internal/app/message"
	tools "arch-agent/internal/app/toolexecutor"
	"context"
)

type RequestContextFactory struct {
	reflector Reflector
}

func NewRequestContextFactory(reflector Reflector) *RequestContextFactory {
	return &RequestContextFactory{
		reflector: reflector,
	}
}

func (f *RequestContextFactory) Build(
	ctx context.Context,
	a AgentConfig,
	memory string,
	messages []message.Message,
	contextDescription string,
	tools []tools.ToolDefinition,
) (*ExecutionContext, error) {

	limit := min(len(messages), 6)

	reflection, err := f.reflector.Reflect(ctx, messages[len(messages)-limit:], a.Personality)
	if err != nil {
		return nil, err
	}

	return NewExecutionContext(reflection, contextDescription, memory, a, tools), nil
}
