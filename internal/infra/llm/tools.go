package llm

import (
	tools "arch-agent/internal/app/toolexecutor"
	"arch-agent/internal/domain/conversation"
	"context"
	"encoding/json"
	"fmt"
)

type Tool struct {
	tools.ToolDefinition
	CallRsolver func(conversation.ToolArguments) (string, error)
}

type ToolCallRecivier struct {
	toolBundle map[string]Tool
}

func NewToolCallRecivier(tools []Tool) *ToolCallRecivier {
	rec := &ToolCallRecivier{
		toolBundle: map[string]Tool{},
	}

	for _, t := range tools {
		rec.toolBundle[t.Name] = t
	}

	return rec
}

func (b *ToolCallRecivier) Tools() []tools.ToolDefinition {
	result := make([]tools.ToolDefinition, 0, len(b.toolBundle))
	for _, t := range b.toolBundle {
		result = append(result, t.ToolDefinition)
	}
	return result
}

func (b *ToolCallRecivier) SendCall(ctx context.Context, toolName string, args conversation.ToolArguments) (string, error) {
	tool, ok := b.toolBundle[toolName]
	if !ok {
		return fmt.Sprintf("error. have no %s", toolName), fmt.Errorf("Tool is not found %s", toolName)
	}
	return tool.CallRsolver(args)
}

// helpers
func WrapArgumentedCallResolver[T any](
	callResolver func(T) (string, error),
) func(conversation.ToolArguments) (string, error) {

	return func(ags conversation.ToolArguments) (string, error) {
		var typedArgs T
		if err := json.Unmarshal(ags, &typedArgs); err != nil {
			return fmt.Sprintf("invalid parameters %s", string(ags)), err
		}
		return callResolver(typedArgs)
	}
}
