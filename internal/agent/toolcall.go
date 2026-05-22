package agent

import (
	"encoding/json"
	"fmt"
)

type ToolArguments []byte

func (r ToolArguments) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

type ToolCall struct {
	// tool call id that is assinged by provider
	ID        string
	ToolName  string
	Arguments ToolArguments
}

func NewToolCall(id string, toolName string, arguments ToolArguments) *ToolCall {
	return &ToolCall{
		ID:        id,
		ToolName:  toolName,
		Arguments: arguments,
	}
}

func (tc *ToolCall) String() string {
	return fmt.Sprintf("tool:%s, args:%s\n", tc.ToolName, string(tc.Arguments))
}

type ToolCallResult struct {
	ID     string
	Result string
}

func NewToolCallResult(id string, result string) ToolCallResult {
	return ToolCallResult{
		ID:     id,
		Result: result,
	}
}
