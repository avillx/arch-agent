package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"fmt"
	"strings"
)

// edit_file
type EditFileTool struct{ fs FS }

func NewEditFileTool(fs FS) *EditFileTool { return &EditFileTool{fs} }

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

	if IsReadOnly(args.Path) {
		return "", fmt.Errorf("this path is read only")
	}

	if !IsTextFile(args.Path) {
		return "", fmt.Errorf("you can edit files only with text extensions")
	}

	internal, err := toInternal(args.Path)
	if err != nil {
		return "", err
	}

	data, err := t.fs.ReadFile(internal)
	if err != nil {
		return "", wrapFSError(err, args.Path)
	}

	content := string(data)
	count := strings.Count(content, args.OldStr)

	switch count {
	case 0:
		return "", fmt.Errorf("%s: old_str not found", args.Path)
	case 1:
		// exactly one match — safe to replace
	default:
		return "", fmt.Errorf("%s: old_str found %d times, must be unique", args.Path, count)
	}

	updated := strings.Replace(content, args.OldStr, args.NewStr, 1)
	if err := t.fs.WriteToFile(internal, []byte(updated)); err != nil {
		return "", wrapFSError(err, args.Path)
	}

	return fmt.Sprintf("edited %s", args.Path), nil
}
