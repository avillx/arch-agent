package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"context"
)

type ReadFileTool struct {
	fs *files.FileSystem
}

func NewReadFileTool(fs *files.FileSystem) *ReadFileTool {
	return &ReadFileTool{fs: fs}
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
			Description: "File path, e.g. './shared/project-x/README.md'",
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

	if args.StartLine != nil || args.EndLine != nil {
		res, err := t.fs.ReadFile(args.Path)

		return extractLines(res, args.StartLine, args.EndLine),
			mapErrs(err)
	}

	data, err := t.fs.ReadFile(args.Path)
	if err != nil {
		return "", mapErrs(err)
	}
	return string(data), nil
}
