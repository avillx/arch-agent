package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const skillMD = "SKILL.md"

type ReadFileTool struct{ fs FS }

func NewReadFileTool(fs FS) *ReadFileTool { return &ReadFileTool{fs} }

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
			Description: "File path, e.g. file:///notes/README.md",
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

	if !IsTextFile(args.Path) {
		return "", fmt.Errorf("you can read files only with text extensions")
	}

	internal, err := toInternal(args.Path)
	if err != nil {
		return "", err
	}

	data, err := t.fs.ReadFile(internal)
	if err != nil {
		return "", wrapFSError(err, args.Path)
	}

	lines := strings.Split(string(data), "\n")
	total := len(lines)

	start, end := 1, total
	if args.StartLine != nil {
		start = *args.StartLine
	}
	if args.EndLine != nil {
		end = *args.EndLine
	}

	start = max(1, min(start, total))
	end = max(start, min(end, total))

	content := strings.Join(lines[start-1:end], "\n")

	if isSkill(args.Path) {
		content = CutFrontmatter(content)
	}

	if start > 1 || end < total {
		return fmt.Sprintf("[lines %d–%d of %d]\n%s", start, end, total, content), nil
	}
	return content, nil
}

var frontmatterRE = regexp.MustCompile(
	`(?s)\A---\r?\n.*?\r?\n---(?:\r?\n|$)`,
)

func CutFrontmatter(file string) string {
	return frontmatterRE.ReplaceAllString(file, "")
}

func isSkill(path string) bool {
	return filepath.Base(path) == skillMD
}
