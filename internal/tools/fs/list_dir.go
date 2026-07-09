package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"context"
	"strings"
)

type ListDirTool struct {
	fs *files.FileSystem
}

func NewListDirTool(
	fs *files.FileSystem,
) *ListDirTool {
	return &ListDirTool{
		fs: fs,
	}
}

func (t *ListDirTool) Name() agent.ToolName { return "list_dir" }

func (t *ListDirTool) Description() string {
	return "List entries in a directory; returns one path per line"
}
func (t *ListDirTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Directory path, e.g. './shared",
		},
	}
}

func (t *ListDirTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Path string `json:"path"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	entries, err := t.fs.ReadDir(args.Path)
	if err != nil {
		return "", mapErrs(err)
	}

	if len(entries) == 0 {
		return "directory is empty", nil
	}

	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = formatEntry(t.fs, args.Path, e)
	}
	return strings.Join(lines, "\n"), nil
}
