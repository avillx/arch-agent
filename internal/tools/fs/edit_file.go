package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"arch-agent/internal/types"
	"context"
	"fmt"
	"strings"
)

type EditFileTool struct {
	fsFactory RuledAccessFactory
	repo      agent.Repo
}

func NewEditFileTool(
	fsFactory RuledAccessFactory,
) *EditFileTool {
	return &EditFileTool{
		fsFactory: fsFactory,
	}
}

func (t *EditFileTool) Name() agent.ToolName { return "edit_file" }
func (t *EditFileTool) Description() string {
	return "replace a unique string in a file; old_str must match exactly once"
}

func (t *EditFileTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "File path",
		},
		{
			Name:        "old_str",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Unique string to replace",
		},
		{
			Name:        "new_str",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Replacement string; empty string to delete",
		},
	}
}

func (t *EditFileTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Path   string `json:"path"`
		OldStr string `json:"old_str"`
		NewStr string `json:"new_str"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	rfs, err := t.fsFactory(ctx)
	if err != nil {
		return "", err
	}

	data, err := rfs.ReadFile(args.Path)
	if err != nil {
		return "", mapErrsToAgentMistake(err)
	}

	content := string(data)
	count := strings.Count(content, args.OldStr)

	switch {
	case count == 0:
		return "", types.NewAgentMistakeErrorf("%s: old_str not found", args.Path)
	case count > 1:
		return "", types.NewAgentMistakeErrorf("%s: old_str found %d times, must be unique", args.Path, count)
	}

	updated := strings.Replace(content, args.OldStr, args.NewStr, 1)
	if err := rfs.WriteFile(args.Path, []byte(updated)); err != nil {
		return "", mapErrsToAgentMistake(err)
	}

	return fmt.Sprintf("edited %s", args.Path), nil
}
