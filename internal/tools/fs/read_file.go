package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
)

type ReadFileTool struct {
	fsFactory RuledAccessFactory
}

func NewReadFileTool(fsFactory RuledAccessFactory) *ReadFileTool {
	return &ReadFileTool{fsFactory: fsFactory}
}

func (t *ReadFileTool) Name() agent.ToolName { return "read_file" }

func (t *ReadFileTool) Description() string {
	return "Read file content, optionally limited to a line range (1-indexed)"
}
func (t *ReadFileTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "File path, e.g. /mnt/notes/README.md",
		},
		{
			Name:        "start_line",
			Required:    false,
			Type:        agent.TypeNumber,
			Description: "First line to read (default: 1)",
		},
		{
			Name:        "end_line",
			Required:    false,
			Type:        agent.TypeNumber,
			Description: "Last line to read (default: end of file)",
		},
	}
}

func (t *ReadFileTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Path      string `json:"path"`
		StartLine *int   `json:"start_line,omitempty"`
		EndLine   *int   `json:"end_line,omitempty"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	rfs, err := t.fsFactory(ctx)
	if err != nil {
		return "", err
	}

	if args.StartLine != nil || args.EndLine != nil {
		res, err := rfs.ReadLines(args.Path, args.StartLine, args.EndLine)
		return res, mapErrsToAgentMistake(err)
	}

	data, err := rfs.ReadFile(args.Path)
	if err != nil {
		return "", mapErrsToAgentMistake(err)
	}
	return string(data), nil
}
