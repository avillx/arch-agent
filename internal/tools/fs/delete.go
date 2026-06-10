package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

// delete_file
type DeleteTool struct{ fs FS }

func NewDeleteTool(fs FS) *DeleteTool { return &DeleteTool{fs} }

func (t *DeleteTool) Name() agent.ToolName { return "delete_file" }
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

	if IsReadOnly(args.Path) {
		return "", fmt.Errorf("this path is read only")
	}

	internal, err := toInternal(args.Path)
	if err != nil {
		return "", err
	}

	if err := t.fs.Delete(internal); err != nil {
		return "", wrapFSError(err, args.Path)
	}

	return fmt.Sprintf("deleted %s", args.Path), nil
}
