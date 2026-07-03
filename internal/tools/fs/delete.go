package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type DeleteTool struct {
	fsFactory RuledAccessFactory
}

func NewDeleteTool(fsFactory RuledAccessFactory) *DeleteTool {
	return &DeleteTool{fsFactory: fsFactory}
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

	rfs, err := t.fsFactory(ctx)
	if err != nil {
		return "", err
	}

	if err := rfs.Delete(args.Path); err != nil {
		return "", mapErrsToAgentMistake(err)
	}

	return fmt.Sprintf("deleted %s", args.Path), nil
}
