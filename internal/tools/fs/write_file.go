package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

// write_file
type WriteFileTool struct{ fs FS }

func NewWriteFileTool(fs FS) *WriteFileTool { return &WriteFileTool{fs} }

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
			Description: "file path, e.g. file:///notes/README.md",
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

	if IsReadOnly(args.Path) {
		return "", fmt.Errorf("this path is read only")
	}

	if !IsTextFile(args.Path) {
		return "", fmt.Errorf("you can write only files with text extensions")
	}

	internal, err := toInternal(args.Path)
	if err != nil {
		return "", err
	}

	data := []byte(args.Content)

	if args.Mode == "append" {
		err = t.fs.AppendToFile(internal, data)
	} else {
		err = t.fs.WriteToFile(internal, data)
	}
	if err != nil {
		return "", wrapFSError(err, args.Path)
	}

	return fmt.Sprintf("wrote %d bytes to %s", len(data), args.Path), nil
}
