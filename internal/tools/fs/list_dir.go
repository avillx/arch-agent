package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"path"
	"strings"
)

// list_dir
type ListDirTool struct{ fs FS }

func NewListDirTool(fs FS) *ListDirTool { return &ListDirTool{fs} }

func (t *ListDirTool) Name() agent.ToolName { return "list_dir" }

func (t *ListDirTool) Description() string {
	return "List entries in a directory; returns one file:/// URI per line"
}
func (t *ListDirTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "directory path, e.g. file:///notes/",
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

	internal, err := toInternal(args.Path)
	if err != nil {
		return "", err
	}

	entries, err := t.fs.ReadDir(internal)
	if err != nil {
		return "", wrapFSError(err, args.Path)
	}

	if len(entries) == 0 {
		return "directory is empty", nil
	}

	lines := make([]string, len(entries))
	for i, name := range entries {
		lines[i] = toAgent(path.Join(internal, name))
	}
	return strings.Join(lines, "\n"), nil
}
