package llm

import (
	"arch-agent/internal/app/types"
	"context"
	"encoding/json"
	"fmt"
)

type Tool struct {
	types.ToolDefinition
	CallRsolver func(types.ToolArguments) (string, error)
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

func (b *ToolCallRecivier) Tools() ([]types.ToolDefinition, error) {
	result := make([]types.ToolDefinition, 0, len(b.toolBundle))
	for _, t := range b.toolBundle {
		result = append(result, t.ToolDefinition)
	}
	return result, nil
}

func (b *ToolCallRecivier) ReciveCall(ctx context.Context, call *types.ToolCall) (string, error) {
	tool, ok := b.toolBundle[call.ToolName()]
	if !ok {
		return fmt.Sprintf("error. have no %s", call.ToolName()), fmt.Errorf("Tool is not found %s", call.ToolName())
	}
	return tool.CallRsolver(call.Arguments())
}

// helpers
func WrapArgumentedCallResolver[T any](
	callResolver func(T) (string, error),
) func(types.ToolArguments) (string, error) {

	return func(ags types.ToolArguments) (string, error) {
		var typedArgs T
		if err := json.Unmarshal(ags, &typedArgs); err != nil {
			return fmt.Sprintf("invalid parameters %s", string(ags)), err
		}
		return callResolver(typedArgs)
	}
}
