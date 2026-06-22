package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type DeleteTool struct {
	fs   *files.FileSystem
	repo agent.Repo
}

func NewDeleteTool(fs *files.FileSystem, repo agent.Repo) *DeleteTool {
	return &DeleteTool{fs: fs, repo: repo}
}

func (t *DeleteTool) Name() agent.ToolName { return "delete" }
func (t *DeleteTool) Description() string {
	return "permanently delete a file or directory; this operation cannot be undone"
}
func (t *DeleteTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Path to the file or directory",
		},
	}
}

func (t *DeleteTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Path string `json:"path"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	rfs, err := newRuledFS(ctx, t.fs, t.repo)
	if err != nil {
		return "", err
	}

	if err := rfs.Delete(args.Path); err != nil {
		return "", err
	}

	return fmt.Sprintf("deleted %s", args.Path), nil
}
