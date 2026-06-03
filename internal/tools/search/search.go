package search

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"fmt"
	"strings"
)

type Result struct {
	Title   string
	Link    string
	Snippet string
}

type WebSearchEngine interface {
	Search(context.Context, string, int) ([]Result, error)
}

type WebSearchTool struct {
	engine WebSearchEngine
}

func NewWebSearchTool(engine WebSearchEngine) *WebSearchTool {
	return &WebSearchTool{
		engine: engine,
	}
}

func (t *WebSearchTool) Name() agent.ToolName {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "searchs web"
}

func (t *WebSearchTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "query",
			Required:    true,
			Type:        agent.TypeString,
			Description: "search query",
		},
		{
			Name:        "results",
			Required:    true,
			Type:        agent.TypeNumber,
			Description: "number of results",
		},
	}
}

func (t *WebSearchTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Query   string `json:"query"`
		Results int    `json:"results"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	results, err := t.engine.Search(ctx, args.Query, args.Results)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Search results for %s \n\n", args.Query))

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("## %s\n", r.Title))
		sb.WriteString(fmt.Sprintf("URL: `%s`\n", r.Link))
		sb.WriteString(fmt.Sprintf("%s\n\n", r.Snippet))
	}

	return sb.String(), nil
}
