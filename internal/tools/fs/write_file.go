package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type WriteFileTool struct {
	fsFactory RuledAccessFactory
}

func NewWriteFileTool(fsFactory RuledAccessFactory) *WriteFileTool {
	return &WriteFileTool{fsFactory: fsFactory}
}

func (t *WriteFileTool) Name() agent.ToolName { return "write_file" }
func (t *WriteFileTool) Description() string {
	return "Write content to a file, creating it if it does not exist; default mode overwrites"
}
func (t *WriteFileTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "file path, e.g. /mnt/notes/README.md",
		},
		{
			Name:        "content",
			Required:    true,
			Type:        agent.TypeString,
			Description: "text content to write",
		},
		{
			Name:        "mode",
			Required:    false,
			Type:        agent.TypeString,
			Description: `"overwrite" (default) or "append"`,
			Enum:        []string{"overwrite", "append"},
		},
	}
}

func (t *WriteFileTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Mode    string `json:"mode"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	rfs, err := t.fsFactory(ctx)
	if err != nil {
		return "", err
	}

	data := []byte(args.Content)

	if args.Mode == "append" {
		err = rfs.AppendToFile(args.Path, data)
	} else {
		err = rfs.WriteFile(args.Path, data)
	}
	if err != nil {
		return "", ruleBreakToAgentMistake(err)
	}

	return fmt.Sprintf("wrote %d bytes to %s", len(data), args.Path), nil
}
