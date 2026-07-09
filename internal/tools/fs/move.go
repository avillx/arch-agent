package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type MoveFileTool struct {
	fs *files.FileSystem
}

func NewMoveFileTool(fs *files.FileSystem) *MoveFileTool {
	return &MoveFileTool{fs: fs}
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
			Description: "Source file path './shared/project-x/README.md'",
		},
		{
			Name:        "dst",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Destination file path './shared/other-project/README.md'",
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

	data, err := t.fs.ReadFile(args.Src)
	if err != nil {
		return "", mapErrs(err)
	}

	if err := t.fs.WriteToFile(args.Dst, data); err != nil {
		return "", mapErrs(err)
	}

	if err := t.fs.Delete(args.Src); err != nil {
		return fmt.Sprintf("file copied to %s but failed to remove source", args.Dst), mapErrs(err)
	}

	return fmt.Sprintf("moved %s → %s", args.Src, args.Dst), nil
}
