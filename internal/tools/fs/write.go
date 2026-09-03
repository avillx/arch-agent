package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type WriteTool struct {
	fs *files.FileSystem
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
			Enum:        []string{"overwrite", "append"},
		},
	}
}

func (t *WriteTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Mode    string `json:"mode"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	data := []byte(args.Content)

	if args.Mode == "append" {
		err = t.fs.AppendToFile(args.Path, data)
	} else {
		err = t.fs.WriteToFile(args.Path, data)
	}
	if err != nil {
		return nil, mapErrs(err)
	}

	return tools.Result(fmt.Sprintf("wrote %d bytes to %s", len(data), args.Path)), nil
}
