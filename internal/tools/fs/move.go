package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type MoveTool struct {
	fs *files.FileSystem
}

func (t *MoveTool) Name() agent.ToolName { return "move" }
func (t *MoveTool) Description() string {
	return "Move or rename a file or directory from src to dst"
}

func (t *MoveTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:        "src",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Source path e.g. './shared/project-x' or './shared/project-x/README.md'",
		},
		{
			Name:        "dst",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Destination path './shared/other-project' or './shared/other-project/README.md'",
		},
	}
}

func (t *MoveTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	if err := t.fs.Rename(args.Src, args.Dst); err != nil {
		return nil, mapErrs(err)
	}

	msg := fmt.Sprintf("moved %s → %s", args.Src, args.Dst)
	return tools.Result(msg), nil
}
