package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type DeleteTool struct {
	fs *files.FileSystem
}

func NewDeleteTool(fs *files.FileSystem) *DeleteTool {
	return &DeleteTool{fs: fs}
}

func (t *DeleteTool) Name() agent.ToolName { return "delete" }
func (t *DeleteTool) Description() string {
	return "permanently delete a file or directory; this operation cannot be undone"
}
func (t *DeleteTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Path to the file or directory './shared/deprecated",
		},
	}
}

func (t *DeleteTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Path string `json:"path"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	if err := t.fs.DeleteAll(args.Path); err != nil {
		return nil, mapErrs(err)
	}

	return tools.Result(fmt.Sprintf("deleted %s", args.Path)), nil
}
