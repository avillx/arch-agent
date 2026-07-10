package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"arch-agent/internal/types"
	"context"
	"path"
	"strings"
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

func (t *ReadFileTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Path      string `json:"path"`
		StartLine *int   `json:"start_line,omitempty"`
		EndLine   *int   `json:"end_line,omitempty"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	// if is image
	if imageType := detectImageType(args.Path); imageType != "" {
		data, err := t.fs.ReadFile(args.Path)
		if err != nil {
			return nil, err
		}

		content, err := agent.NewImageContent(imageType, data)
		if err != nil {
			return nil, types.NewAgentMistakeError(err.Error())
		}

		return []agent.ContentPart{content}, nil
	}

	if args.StartLine != nil || args.EndLine != nil {
		res, err := t.fs.ReadFile(args.Path)
		lines := extractLines(res, args.StartLine, args.EndLine)

		return tools.Result(lines), mapErrs(err)
	}

	data, err := t.fs.ReadFile(args.Path)
	if err != nil {
		return nil, mapErrs(err)
	}
	return tools.Result(string(data)), nil
}

func detectImageType(p string) agent.AllowedMIME {

	switch strings.TrimPrefix(path.Ext(p), ".") {
	case "png":
		return agent.Png
	case "bmp":
		return agent.Bmp
	case "jpeg", "jpg":
		return agent.Jpeg
	case "webp":
		return agent.Webp
	default:
		return ""
	}
}
