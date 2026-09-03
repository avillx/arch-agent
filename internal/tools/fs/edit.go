package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"arch-agent/internal/types"
	"context"
	"fmt"
	"strings"
)

type EditTool struct {
	fs *files.FileSystem
}

func (t *EditTool) Name() agent.ToolName { return "edit" }
func (t *EditTool) Description() string {
	return "Replace a unique string in a file; 'old' string must match exactly once"
}

func (t *EditTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "File path e.g './shared/project-x/README.md'",
		},
		{
			Name:        "old",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Unique string to replace",
		},
		{
			Name:        "new",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Replacement string; empty string to delete",
		},
	}
}

func (t *EditTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Path string `json:"path"`
		Old  string `json:"old"`
		New  string `json:"new"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	data, err := t.fs.ReadFile(args.Path)
	if err != nil {
		return nil, mapErrs(err)
	}

	content := string(data)
	count := strings.Count(content, args.Old)

	switch {
	case count == 0:
		return nil, types.NewAgentMistakeErrorf("%s: 'old' not found", args.Path)
	case count > 1:
		return nil, types.NewAgentMistakeErrorf("%s: 'old' found %d times, must be unique", args.Path, count)
	}

	updated := strings.Replace(content, args.Old, args.New, 1)
	if err := t.fs.WriteToFile(args.Path, []byte(updated)); err != nil {
		return nil, mapErrs(err)
	}

	return tools.Result(fmt.Sprintf("edited %s", args.Path)), nil
}
