package conversation

import "encoding/json"

type ToolArguments json.RawMessage

// bounded context value object
type ToolCall struct {
	// tool call id that is assinged by provider
	id        string
	toolName  string
	arguments ToolArguments
}

func (c *ToolCall) ID() string {
	return c.id
}

func (c *ToolCall) Arguments() ToolArguments {
	return c.arguments
}

func (c *ToolCall) ToolName() string {
	return c.toolName
}

func NewToolCall(id string, toolName string, arguments ToolArguments) *ToolCall {
	return &ToolCall{
		id:        id,
		toolName:  toolName,
		arguments: arguments,
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
