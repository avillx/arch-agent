package mcprecivier

import (
	"arch-agent/internal/app/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func pullTools(session *mcp.ClientSession) ([]types.ToolDefinition, error) {
	// TODO: remove background and use child from root context
	toolList, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		return nil, err
	}
	return toInteranlTools(toolList.Tools), nil
}

func toInteranlTools(mcpTools []*mcp.Tool) []types.ToolDefinition {
	toolDefs := []types.ToolDefinition{}
	for _, t := range mcpTools {
		toolDefs = append(toolDefs, toInternalTool(t))
	}
	return toolDefs
}

func toInternalTool(t *mcp.Tool) types.ToolDefinition {
	return types.ToolDefinition{
		Name:        t.Name,
		Description: t.Description,
		Properties:  toInternalProperties(t.InputSchema),
	}
}

func castMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func toInternalProperties(schema any) []types.ToolProperty {
	m, ok := castMap(schema)
	if !ok {
		return nil
	}

	required := map[string]bool{}
	if req, ok := m["required"].([]any); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				required[s] = true
			}
		}
	}

	propsMap, ok := castMap(m["properties"])
	if !ok {
		return nil
	}

	result := make([]types.ToolProperty, 0, len(propsMap))
	for name, prop := range propsMap {
		propMap, ok := castMap(prop)
		if !ok {
			continue
		}
		result = append(result, toInternalProperty(name, propMap, required[name]))
	}
	return result
}

func toInternalProperty(name string, prop map[string]any, required bool) types.ToolProperty {
	p := types.ToolProperty{Name: name, Required: required}
	if t, ok := prop["type"].(string); ok {
		p.Type = types.PropertyType(t)
	}
	if d, ok := prop["description"].(string); ok {
		p.Description = d
	}
	if enum, ok := prop["enum"].([]any); ok {
		for _, e := range enum {
			if s, ok := e.(string); ok {
				p.Enum = append(p.Enum, s)
			}
		}
	}
	return p
}

func toCallToolParams(call *types.ToolCall) (*mcp.CallToolParams, error) {
	var args any
	if err := json.Unmarshal(call.Arguments, &args); err != nil {
		return nil, fmt.Errorf("unmarshal arguments: %w", err)
	}
	p := &mcp.CallToolParams{
		Name:      call.ToolName,
		Arguments: args,
	}
	return p, nil
}

func resultToString(res *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, c := range res.Content {
		if text, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(text.Text)
		}
	}
	return sb.String()
}

func toMCPTool(def types.ToolDefinition) mcp.Tool {
	return mcp.Tool{
		Name:        def.Name,
		Description: def.Description,
		InputSchema: toMCPSchema(def.Properties),
	}
}

func toMCPSchema(props []types.ToolProperty) *jsonschema.Schema {
	schema := &jsonschema.Schema{
		Type:       "object",
		Properties: map[string]*jsonschema.Schema{},
	}
	for _, p := range props {
		schema.Properties[p.Name] = toMCPSchemaProp(p)
		if p.Required {
			schema.Required = append(schema.Required, p.Name)
		}
	}
	return schema
}

func toMCPSchemaProp(p types.ToolProperty) *jsonschema.Schema {
	s := &jsonschema.Schema{
		Type:        string(p.Type),
		Description: p.Description,
	}
	for _, e := range p.Enum {
		s.Enum = append(s.Enum, e)
	}
	return s
}

func WrapForMCP(callResolver func(types.ToolArguments) (string, error)) mcp.ToolHandler {
	return func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		req.Params.Meta.GetMeta()
		args := types.ToolArguments(req.Params.Arguments)
		value, err := callResolver(args)
		return CreateResult(value, err), nil
	}
}

func CreateResult(value string, err error) *mcp.CallToolResult {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: value}},
	}
	if err != nil {
		result.SetError(err)
	}
	return result
}

func createToolMap(defs []types.ToolDefinition) map[string]types.ToolDefinition {
	toolMap := make(map[string]types.ToolDefinition, len(defs))
	for _, def := range defs {
		toolMap[def.Name] = def
	}
	return toolMap
}

func extractAgentPrompt(session *mcp.ClientSession, promptName string) (string, error) {
	promptsResult, err := session.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name: promptName,
	})
	if err != nil {
		return "", err
	}

	if len(promptsResult.Messages) > 0 {
		if textContent, ok := promptsResult.Messages[0].Content.(*mcp.TextContent); ok {
			return textContent.Text, nil
		}
		return "", fmt.Errorf("mcp server %s return bad prompt %s", session.ID(), promptName)
	}

	return "", errors.Join(ErrNoMCPPrompt, fmt.Errorf("server - %s \n prompt name - %s", session.ID(), promptName))
}
