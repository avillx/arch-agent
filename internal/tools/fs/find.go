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
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

var lineBreaker = []byte("\n")

type FindTool struct {
	fs           *files.FileSystem
	skipPatterns []string
}

func NewFindTool(fs *files.FileSystem, skipPatterns []string) (*FindTool, error) {

	// validate skip patterns
	for _, p := range skipPatterns {
		if !doublestar.ValidatePattern(p) {
			return nil, fmt.Errorf("invalid pattern '%s'", p)
		}
	}

	return &FindTool{
		fs:           fs,
		skipPatterns: skipPatterns,
	}, nil
}

func (t *FindTool) Name() agent.ToolName { return "find" }

func (t *FindTool) Description() string {
	return "glob-like path search; When regex specified returns matched file content (grep-like behaviour)"
}

func (t *FindTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:        "glob",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Glob pattern e.g. **/**.md",
		},
		{
			Name:     "regex",
			Required: false,
			Type:     agent.TypeString,
			Description: `Go-compitable regular expression, 
			serve to find content in files; Adds matched file entry to results`,
		},
	}
}

func (t *FindTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Glob  string  `json:"glob"`
		Regex *string `json:"regex"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	var regex *regexp.Regexp
	if args.Regex != nil {
		regex, err = regexp.Compile(*args.Regex)
		if err != nil {
			return nil, types.NewAgentMistakeError("bad regex, allowed only Golang regexp syntax")
		}
	}

	var sb strings.Builder
	err = doublestar.GlobWalk(t.fs, args.Glob, func(path string, d fs.DirEntry) error {

		if d.IsDir() {
			return nil
		}

		path, err := t.fs.ToAbs(path)
		if err != nil {
			return err
		}

		// should skip
		for _, pattern := range t.skipPatterns {
			if doublestar.PathMatchUnvalidated(pattern, path) {
				return nil
			}
		}

		if regex != nil {

			data, err := t.fs.ReadFile(path)
			if err != nil {
				fmt.Fprintf(&sb, "%s err: %v", path, err)
				return nil
			}

			matches := regex.FindAllIndex(data, -1)
			if len(matches) <= 0 {
				return nil
			}

			fmt.Fprintf(&sb, "%s\n", path)
			for _, match := range regex.FindAllIndex(data, -1) {
				lineNumber, lineContent := lineAt(data, match[0])
				fmt.Fprintf(&sb, "line: %d | %s\n", lineNumber, lineContent)
			}
			sb.WriteString("\n")
			return nil
		}

		fmt.Fprintf(&sb, "%s\n", path)

		return nil
	}, doublestar.WithCaseInsensitive())

	if err != nil {
		return nil, mapErrs(err)
	}

	return tools.Result(sb.String()), nil
}

func lineAt(data []byte, idx int) (int, string) {

	start := bytes.LastIndex(data[:idx], lineBreaker)

	// to exclude start index of "\n" e.g. "\nmatch" -> "match"
	// also it replace -1 to 0. no match -> zero index (start in first line)
	start++

	end := bytes.Index(data[start:], lineBreaker)
	if end == -1 {
		// to avoid -1 if it is last line
		end = len(data)
	} else {
		// cause endline is relative to data[startLineIdx:]
		// start line is guranteed > 0
		end += start
	}

	lineNumber := bytes.Count(data[:idx], lineBreaker)
	lineContent := string(data[start:end])

	return lineNumber, lineContent
}
