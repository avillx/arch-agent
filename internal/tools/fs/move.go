package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

// move_file
type MoveFileTool struct{ fs FS }

func NewMoveFileTool(fs FS) *MoveFileTool { return &MoveFileTool{fs} }

func (t *MoveFileTool) Name() agent.ToolName { return "move_file" }
func (t *MoveFileTool) Description() string {
	return "move or rename a file from src to dst"
}
func (t *MoveFileTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "src",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Source file path",
		},
		{
			Name:        "dst",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Destination file path",
		},
	}
}

func (t *MoveFileTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	if IsReadOnly(args.Src) {
		return "", fmt.Errorf("this path is read only")
	}

	if !IsTextFile(args.Src) && IsTextFile(args.Dst) {
		return "", fmt.Errorf("can't change non text file extension to text")
	}

	srcInternal, err := toInternal(args.Src)
	if err != nil {
		return "", err
	}
	dstInternal, err := toInternal(args.Dst)
	if err != nil {
		return "", err
	}

	data, err := t.fs.ReadFile(srcInternal)
	if err != nil {
		return "", wrapFSError(err, args.Src)
	}

	if err := t.fs.WriteToFile(dstInternal, data); err != nil {
		return "", wrapFSError(err, args.Dst)
	}

	if err := t.fs.Delete(srcInternal); err != nil {
		// dst was written successfully; src still exists — inform the agent
		return "", fmt.Errorf("file copied to %s but %s", args.Dst, wrapFSError(err, args.Src))
	}

	return fmt.Sprintf("moved %s → %s", args.Src, args.Dst), nil
}
