package mcp

import (
	"arch-agent/internal/agent"
	"context"
	"encoding/json"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var _ agent.Tool = (*mcpTool)(nil)

type mcpTool struct {
	name        agent.ToolName
	description string
	schema      []agent.ToolProperty
	session     *mcpsdk.ClientSession
}

func (t *mcpTool) Name() agent.ToolName         { return t.name }
func (t *mcpTool) Description() string          { return t.description }
func (t *mcpTool) Schema() []agent.ToolProperty { return t.schema }
func (t *mcpTool) Call(ctx context.Context, args agent.ToolArguments) (string, error) {
	var m map[string]any
	if err := json.Unmarshal(args, &m); err != nil {
		return "", fmt.Errorf("mcp tool %s: %w", t.name, err)
	}

	result, err := t.session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      string(t.name),
		Arguments: m,
	})
	if err != nil {
		return "", fmt.Errorf("mcp tool %s: %w", t.name, err)
	}

	return textContent(result.Content), nil
}
