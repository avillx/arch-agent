package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type InternalTool struct {
	types.ToolDefinition
	CallRsolver func(types.ToolArguments, string) (string, error)
}

func WrapArgumentedCallResolver[T any](
	callResolver func(T, string) (string, error),
) func(types.ToolArguments, string) (string, error) {

	return func(ags types.ToolArguments, sign string) (string, error) {
		var typedArgs T
		if err := json.Unmarshal(ags, &typedArgs); err != nil {
			return fmt.Sprintf("invalid parameters %s", string(ags)), err
		}
		return callResolver(typedArgs, sign)
	}
}

type InternalServer struct {
	name         string
	instructions func(agentID agent.ID) string
	toolMap      map[string]*InternalTool
}

func NewInternalServer(name string, intructions func(agentID agent.ID) string, internalTools ...*InternalTool) *InternalServer {
	toolMap := map[string]*InternalTool{}
	for _, t := range internalTools {
		toolMap[t.Name] = t
	}

	return &InternalServer{
		name:         name,
		instructions: intructions,
		toolMap:      toolMap,
	}
}

func (s *InternalServer) Name() string {
	return s.name
}

func (s *InternalServer) ToolGuide(agentID agent.ID) string {
	return s.instructions(agentID)
}

func (s *InternalServer) Tools() []types.ToolDefinition {
	defs := []types.ToolDefinition{}
	for _, tool := range s.toolMap {
		defs = append(defs, tool.ToolDefinition)
	}
	return defs
}

func (s *InternalServer) SendCall(ctx context.Context, call *types.ToolCall, sign string) (string, error) {
	tool, ok := s.toolMap[call.ToolName]
	if !ok {
		return "",
			errors.Join(
				types.ErrBadToolCall,
				fmt.Errorf("tool %s not found in server %s", call.ToolName, s.Name()),
			)
	}

	return tool.CallRsolver(call.Arguments, sign)
}
