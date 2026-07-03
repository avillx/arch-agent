package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"context"
	"fmt"
	"os"
	"path"
	"strings"
)

type ListDirTool struct {
	fsFactory RuledAccessFactory
}

func NewListDirTool(
	fsFactory RuledAccessFactory,
) *ListDirTool {
	return &ListDirTool{
		fsFactory: fsFactory,
	}
}

func (t *ListDirTool) Name() agent.ToolName { return "list_dir" }

func (t *ListDirTool) Description() string {
	return "List entries in a directory; returns one /mnt/ URI per line"
}
func (t *ListDirTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "directory path, e.g. /mnt/notes/",
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

	rfs, err := t.fsFactory(ctx)
	if err != nil {
		return "", err
	}

	entries, err := rfs.ReadDir(args.Path)
	if err != nil {
		return "", mapErrsToAgentMistake(err)
	}

	if len(entries) == 0 {
		return "directory is empty", nil
	}

	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = formatEntry(rfs, args.Path, e)
	}
	return strings.Join(lines, "\n"), nil
}

func formatEntry(rfs interface {
	ReadFile(path string) ([]byte, error)
}, dirPath string, e os.DirEntry) string {
	label := path.Join(dirPath, e.Name())

	info, err := e.Info()
	if err != nil {
		return label
	}
	size := files.FormatSize(int(info.Size()))

	if e.IsDir() {
		return fmt.Sprintf("%s %s", label, size)
	}

	content, err := rfs.ReadFile(path.Join(dirPath, e.Name()))
	if err != nil {
		return fmt.Sprintf("%s %s", label, size)
	}

	lineCount := strings.Count(string(content), "\n")
	if len(content) > 0 && content[len(content)-1] != '\n' {
		lineCount++
	}
	return fmt.Sprintf("%s %s [%d lines]", label, size, lineCount)
}
