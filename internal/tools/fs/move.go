package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type MoveFileTool struct {
	fs   *files.FileSystem
	repo agent.Repo
}

func NewMoveFileTool(fs *files.FileSystem, repo agent.Repo) *MoveFileTool {
	return &MoveFileTool{fs: fs, repo: repo}
}

func (t *MoveFileTool) Name() agent.ToolName { return "move_file" }
func (t *MoveFileTool) Description() string {
	return "move or rename a file from src to dst"
}
func (t *MoveFileTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "src",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Source file path",
		},
		{
			Name:        "dst",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Destination file path",
		},
	}
}

func (t *MoveFileTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	rfs, err := newRuledFS(ctx, t.fs, t.repo)
	if err != nil {
		return "", err
	}

	data, err := rfs.ReadFile(args.Src)
	if err != nil {
		return "", err
	}

	if err := rfs.WriteFile(args.Dst, data); err != nil {
		return "", err
	}

	if err := rfs.Delete(args.Src); err != nil {
		return "", fmt.Errorf("file copied to %s but failed to remove source: %s", args.Dst, err)
	}

	return fmt.Sprintf("moved %s → %s", args.Src, args.Dst), nil
}
