package types

import "encoding/json"

type ToolArguments []byte

func (r ToolArguments) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

// bounded context value object
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
