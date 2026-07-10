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
	ToolName  ToolName
	Arguments ToolArguments
}

func NewToolCall(id string, toolName ToolName, arguments ToolArguments) *ToolCall {
	return &ToolCall{
		ID:        id,
		ToolName:  toolName,
		Arguments: arguments,
	}
}

func (tc *ToolCall) String() string {
	return fmt.Sprintf("tool:%s, args:%s\n", tc.ToolName, string(tc.Arguments))
}

type ToolResult struct {
	ID     string
	Result []ContentPart
}

func NewToolResult[T string | []ContentPart](id string, content T) *ToolResult {

	return &ToolResult{
		ID:     id,
		Result: NewContent(content),
	}
}
