package shared

import "encoding/json"

type ToolCall struct {
	// tool call id that is assinged by provider
	ID        string
	ToolName  string
	Arguments json.RawMessage
}

func NewToolCall(id string, toolName string, arguments json.RawMessage) *ToolCall {
	return &ToolCall{
		ID:        id,
		ToolName:  toolName,
		Arguments: arguments,
	}
}
