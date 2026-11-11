package fstools

import (
	"ard/internal/agent"
	"ard/internal/files"
	"ard/internal/tools"
	"ard/internal/types"
	"context"
	"fmt"
	"strings"
)

type EditTool struct {
	storage files.FileStorage
}

func (t *EditTool) Name() agent.ToolName { return "edit" }
func (t *EditTool) Description() string {
	return "Replace a unique string in a file; 'old' string must match exactly once"
}

func (t *EditTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "File path e.g 'shared/project-x/README.md'",
			},
			"old": map[string]any{
				"type":        "string",
				"description": "Unique string to replace",
			},
			"new": map[string]any{
				"type":        "string",
				"description": "Replacement string; empty string to delete",
			},
		},
		"required": []string{"path", "old", "new"},
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

	data, err := t.storage.ReadFile(args.Path)
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
	if err := t.storage.WriteFile(args.Path, []byte(updated), files.ModeFilePerm); err != nil {
		return nil, mapErrs(err)
	}

	return tools.Result(fmt.Sprintf("edited %s", args.Path)), nil
}
