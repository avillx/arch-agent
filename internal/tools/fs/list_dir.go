package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"
)

// list_dir
type ListDirTool struct{ fs FS }

func NewListDirTool(fs FS) *ListDirTool { return &ListDirTool{fs} }

func (t *ListDirTool) Name() agent.ToolName { return "list_dir" }

func (t *ListDirTool) Description() string {
	return "List entries in a directory; returns one file:/// URI per line"
}
func (t *ListDirTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "directory path, e.g. file:///notes/",
		},
	}
}

func (t *ListDirTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Path string `json:"path"`
	}](rawArgs)
	if err != nil {
		return "", err
	}
	internal, err := toInternal(args.Path)
	if err != nil {
		return "", err
	}
	entries, err := t.fs.ReadDir(internal)
	if err != nil {
		return "", wrapFSError(err, args.Path)
	}

	if len(entries) == 0 {
		return "directory is empty", nil
	}

	lines := make([]string, len(entries))
	for i, name := range entries {
		filePath := path.Join(internal, name)
		lines[i] = formatEntry(t.fs, filePath)
	}
	return strings.Join(lines, "\n"), nil
}

func formatEntry(fs FS, filePath string) string {
	label := toAgent(filePath)

	content, err := fs.ReadFile(filePath)
	if err != nil {
		return label // директория или нет доступа — просто путь
	}

	size := formatSize(len(content))

	if !IsTextFile(filePath) {
		return fmt.Sprintf("%s %s", label, size)
	}

	lineCount := bytes.Count(content, []byte("\n"))
	if len(content) > 0 && content[len(content)-1] != '\n' {
		lineCount++
	}
	return fmt.Sprintf("%s %s [%d lines]", label, size, lineCount)
}

func formatSize(n int) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1fmb", float64(n)/1024/1024)
	case n >= 1024:
		return fmt.Sprintf("%.1fkb", float64(n)/1024)
	default:
		return fmt.Sprintf("%db", n)
	}
}
