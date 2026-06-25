package mcp

import (
	"arch-agent/internal/agent"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func mcptoolToInternal(tool *mcpsdk.Tool, session *mcpsdk.ClientSession) *mcpTool {
	return &mcpTool{
		name:        agent.ToolName(tool.Name),
		description: tool.Description,
		schema:      schemaProperties(tool.InputSchema),
		session:     session,
	}
}

func schemaProperties(raw any) []agent.ToolProperty {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	required := parseRequired(m["required"])
	props, _ := m["properties"].(map[string]any)
	if props == nil {
		return nil
	}

	out := make([]agent.ToolProperty, 0, len(props))
	for name, raw := range props {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		tp := agent.ToolProperty{Name: name, Required: required[name]}

		v, ok := p["type"].(string)
		if !ok {
			continue
		}
		tp.Type = propertyType(v)

		if v, ok := p["description"].(string); ok {
			tp.Description = v
		}
		if v, ok := p["enum"].([]any); ok {
			for _, e := range v {
				if s, ok := e.(string); ok {
					tp.Enum = append(tp.Enum, s)
				}
			}
		}

		out = append(out, tp)
	}

	return out
}

func parseRequired(v any) map[string]bool {
	req, ok := v.([]any)
	if !ok {
		return nil
	}
	m := make(map[string]bool, len(req))
	for _, r := range req {
		if s, ok := r.(string); ok {
			m[s] = true
		}
	}
	return m
}

func propertyType(t string) agent.PropertyType {
	switch t {
	case "integer", "number":
		return agent.TypeNumber
	case "boolean":
		return agent.TypeBoolean
	default:
		return agent.TypeString
	}
}

func textContent(content []mcpsdk.Content) string {
	var text string
	for _, c := range content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			text += tc.Text
		}
	}
	return text
}
