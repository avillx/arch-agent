package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"arch-agent/internal/types"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type ReadTool struct {
	fs           *files.FileSystem
	skipPatterns []string
}

func NewReadTool(fs *files.FileSystem, skipPatterns []string) (*ReadTool, error) {

	// validate skip patterns
	for _, p := range skipPatterns {
		if !doublestar.ValidatePattern(p) {
			return nil, fmt.Errorf("invalid pattern '%s'", p)
		}
	}

	return &ReadTool{
		fs:           fs,
		skipPatterns: skipPatterns,
	}, nil
}

func (t *ReadTool) Name() agent.ToolName { return "read" }

func (t *ReadTool) Description() string {
	return `Read text files, image files and directories, text files optionally 
limited to a line range, reads dir structure in 2 levels depth`
}

func (t *ReadTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Path to file or directory, e.g. `./shared` './shared/project-x/README.md', './shared/img.png'",
		},
		{
			Name:        "start_line",
			Required:    false,
			Type:        agent.TypeNumber,
			Description: "First line of text file to read (default: 1)",
		},
		{
			Name:        "end_line",
			Required:    false,
			Type:        agent.TypeNumber,
			Description: "Last line of text file to read (default: end of file)",
		},
	}
}

// read file
func (t *ReadTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Path      string `json:"path"`
		StartLine *int   `json:"start_line,omitempty"`
		EndLine   *int   `json:"end_line,omitempty"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	// read directory
	info, err := t.fs.Info(args.Path)
	if err != nil {
		return nil, mapErrs(err)
	}
	if info.IsDir() {
		repr, err := t.ReadDir(args.Path)
		if err != nil {
			return nil, mapErrs(err)
		}
		return tools.Result(repr), nil
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

	text, err := t.ReadTextFile(args.Path, args.StartLine, args.EndLine)
	if err != nil {
		return nil, mapErrs(err)
	}
	return tools.Result(text), nil
}

func (t *ReadTool) ReadTextFile(p string, start, end *int) (string, error) {
	if start != nil || end != nil {
		res, err := t.fs.ReadFile(p)
		lines := extractLines(res, start, end)
		return lines, mapErrs(err)
	}

	data, err := t.fs.ReadFile(p)
	if err != nil {
		return "", mapErrs(err)
	}

	if isBinary(data) {
		return "", types.NewAgentMistakeError("you can't read binary file")
	}

	return string(data), nil
}

func (t *ReadTool) ReadDir(p string) (string, error) {
	absPath, err := t.fs.ToAbs(p)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	rootParts := strings.Split(absPath, string(os.PathSeparator))
	err = t.fs.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {

		// should skip
		for _, pattern := range t.skipPatterns {
			if doublestar.PathMatchUnvalidated(pattern, path) {
				return nil
			}
		}

		labels := []string{}

		// error label
		if err != nil {
			fmt.Fprintf(&sb, "%s [err: %v]\n", path, err)
			return nil
		}

		// directory label
		if d.IsDir() {

			// skip if depth greather than 2 levels
			parts := strings.Split(path, string(os.PathSeparator))
			if len(parts) > len(rootParts)+2 {
				return fs.SkipDir
			}
			labels = append(labels, "[dir]")
		}

		// size label
		info, err := d.Info()
		if err == nil {
			size := int(info.Size())
			labels = append(labels, files.FormatSize(size))
		}

		data, err := t.fs.ReadFile(path)
		if err == nil {
			if isBinary(data) {

				// Binary file add binary label
				labels = append(labels, "[bin]")
			} else {

				// Text file add a lines label
				lineCount := bytes.Count(data, []byte("\n"))
				if len(data) > 0 && data[len(data)-1] != '\n' {
					lineCount++
				}
				labels = append(labels, fmt.Sprintf("[%d lines]", lineCount))
			}
		}

		fmt.Fprintf(&sb, "%s %s\n", path, strings.Join(labels, " "))

		return nil
	})
	if err != nil {
		return "", err
	}

	return sb.String(), nil
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

func isBinary(data []byte) bool {
	return bytes.IndexByte(data, 0x00) != -1
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
