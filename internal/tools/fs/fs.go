package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/prompt"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
)

type instrucredRead struct {
	*ReadFileTool
}

func (r *instrucredRead) AgentInstruction(agt agent.Agent) string {
	return prompt.GetFileSystemInstructionPrompt(r.fs.Cwd(), agt.ID())
}

func WithInstruction(t *ReadFileTool) *instrucredRead {
	return &instrucredRead{ReadFileTool: t}
}

func matchLines(agentPath, content, query string, limit int) []string {
	lower := strings.ToLower(query)
	var matches []string
	for i, line := range strings.Split(content, "\n") {
		if len(matches) >= limit {
			break
		}
		if strings.Contains(strings.ToLower(line), lower) {
			matches = append(matches, fmt.Sprintf("%s:%d: %s", agentPath, i+1, strings.TrimSpace(line)))
		}
	}
	return matches
}

func mapErrs(err error) error {
	if errors.Is(err, types.ErrIsNotExist) {
		return types.NewAgentMistakeError("path is not found, ensure path existence")
	}
	return err
}

func extractLines(data []byte, from, to *int) string {
	lines := strings.Split(string(data), "\n")
	total := len(lines)

	startLine := 1
	endLine := total

	if from != nil {
		startLine = *from
	}
	if to != nil {
		endLine = *to
	}

	startLine = max(1, min(startLine, total))
	endLine = max(startLine, min(endLine, total))

	return strings.Join(lines[startLine-1:endLine], "\n")
}

func formatEntry(
	fs interface {
		ReadFile(path string) ([]byte, error)
	},
	dirPath string,
	e os.DirEntry,
) string {
	label := path.Join(dirPath, e.Name())

	info, err := e.Info()
	if err != nil {
		return label
	}
	size := files.FormatSize(int(info.Size()))

	if e.IsDir() {
		return fmt.Sprintf("%s %s", label, size)
	}

	content, err := fs.ReadFile(path.Join(dirPath, e.Name()))
	if err != nil {
		return fmt.Sprintf("%s %s", label, size)
	}

	lineCount := strings.Count(string(content), "\n")
	if len(content) > 0 && content[len(content)-1] != '\n' {
		lineCount++
	}
	return fmt.Sprintf("%s %s [%d lines]", label, size, lineCount)
}
