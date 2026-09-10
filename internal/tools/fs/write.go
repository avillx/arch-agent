package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type WriteMode string

const (
	Append    WriteMode = "append"
	Overwrite WriteMode = "overwrite"
)

type WriteTool struct {
	storage files.FileStorage
}

func (t *WriteTool) Name() agent.ToolName { return "write" }
func (t *WriteTool) Description() string {
	return "Write content to a file, creating it if it does not exist; default mode overwrites"
}
func (t *WriteTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:     "path",
			Required: true,
			Type:     agent.TypeString,
			Description: `File path, create full path when it is not exist.
e.g. './shared/project-x/README.md', './shared/non/exist/path/README.md'`,
		},
		{
			Name:        "content",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Text content to write",
		},
		{
			Name:        "mode",
			Required:    false,
			Type:        agent.TypeString,
			Description: `"Overwrite" (default) or "append"`,
			Enum:        []string{string(Overwrite), string(Append)},
		},
	}
}

func (t *WriteTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Path    string    `json:"path"`
		Content string    `json:"content"`
		Mode    WriteMode `json:"mode"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	data := []byte(args.Content)

	if args.Mode == Append {
		f, err := t.storage.OpenFile(
			args.Path,
			files.O_APPEND|files.O_WRONLY|files.O_CREATE,
			files.ModeFilePerm,
		)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		if _, err := f.Write(data); err != nil {
			return nil, mapErrs(err)
		}

		msg := fmt.Sprintf("%d bytes appened to %s", len(data), args.Path)
		return tools.Result(msg), nil
	}

	if err := t.storage.WriteFile(args.Path, data, files.ModeFilePerm); err != nil {
		return nil, mapErrs(err)
	}

	msg := fmt.Sprintf("wrote %d bytes to %s", len(data), args.Path)
	return tools.Result(msg), nil
}
