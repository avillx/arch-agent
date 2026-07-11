package mcp

import (
	"arch-agent/internal/agent"
	"fmt"

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

func schemaProperties(raw any) map[string]any {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func toResult(content *mcpsdk.CallToolResult) ([]agent.ContentPart, error) {
	internalContent := []agent.ContentPart{}
	for _, c := range content.Content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			internalContent = append(internalContent, agent.NewContent(tc.Text)...)
		}
		if tc, ok := c.(*mcpsdk.ImageContent); ok {
			contentPart, err := agent.NewImageContent(agent.AllowedMIME(tc.MIMEType), tc.Data)
			if err != nil {
				// unsupported type
				contentPart.Text = fmt.Sprintf("bad content: %s", err.Error())
			}

			internalContent = append(internalContent, contentPart)
		}
	}

	return internalContent, content.GetError()
}
