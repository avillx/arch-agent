package hooks

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/tools"
	fstools "arch-agent/internal/tools/fs"
	"arch-agent/internal/types"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	doublestar "github.com/bmatcuk/doublestar/v4"
)

const unixSeparator = "/"

var readFileToolName = (&fstools.ReadTool{}).Name()   // read_file
var editFileToolName = (&fstools.EditTool{}).Name()   // edit_file
var moveFileToolName = (&fstools.MoveTool{}).Name()   // move_file
var writeFileToolName = (&fstools.WriteTool{}).Name() // write_file
var errIsNotFileTool = errors.New("is not file tools")

type Access int

const (
	No Access = iota
	Read
	Write
)

type Rule struct {
	Pattern string
	Access  Access
}

var _ runtime.ToolCallHook = (*FileAccessHook)(nil)

type FileAccessHook struct {
	rules []Rule
	cwd   string
}

// First rule match wins, so be accuracy with order
func NewFileAccessHook(cwd string, rules ...Rule) (*FileAccessHook, error) {

	// validate patterns
	for _, r := range rules {
		if !doublestar.ValidatePattern(r.Pattern) {
			return nil, fmt.Errorf("invalid pattern '%s'", r.Pattern)
		}
	}

	return &FileAccessHook{
		rules: rules,
		cwd:   cwd,
	}, nil
}

func (h *FileAccessHook) Apply(ctx context.Context, tc *agent.ToolCall) (*agent.ToolCall, error) {

	paths, err := resolvePaths(tc)
	if err != nil {
		// if not file tool then just ignore
		if errors.Is(err, errIsNotFileTool) {
			return tc, nil
		}
		return nil, err
	}

	for _, p := range paths {
		if err := h.verifyPath(tc.ToolName, path.Clean(p)); err != nil {
			return nil, err
		}
	}

	return tc, nil
}

func (h *FileAccessHook) verifyPath(toolName agent.ToolName, p string) error {

	// /skills/test_note.md is abs for this validate
	if !filepath.IsAbs(p) {
		p = filepath.Join(h.cwd, p)
	}

	// deny symlinks
	foundSymlink, err := containsSymlink(p)
	if err != nil {
		return types.NewAgentMistakeError("bad path")
	}
	if foundSymlink {
		return types.NewAgentMistakeError("path contain symlink: symlink is not allowed")
	}

	// ensure access
	access := No
	for _, r := range h.rules {
		// first match first wins
		if match, _ := doublestar.PathMatch(r.Pattern, p); match {
			access = r.Access
			break
		}
	}

	if !isAllow(toolName, access) {
		exp := "access denied"

		if access > No {
			exp = "read only access"
		}

		return types.NewAgentMistakeErrorf("%s: %s", p, exp)
	}

	return nil
}

func isAllow(toolName agent.ToolName, a Access) bool {
	switch toolName {
	case readFileToolName:
		return a > No
	case editFileToolName, writeFileToolName, moveFileToolName:
		return a > Read
	default:
		return false
	}
}

func containsSymlink(p string) (bool, error) {
	cur := filepath.Clean(p)
	for {
		resolved, err := filepath.EvalSymlinks(cur)
		if err == nil {
			return resolved != cur, nil
		}
		if !os.IsNotExist(err) {
			return false, err
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return false, nil
		}
		cur = parent
	}
}

func resolvePaths(tc *agent.ToolCall) ([]string, error) {

	switch tc.ToolName {
	case
		readFileToolName,
		editFileToolName,
		writeFileToolName:

		args, err := tools.UnwrapArgs[struct {
			Path string `json:"path"`
		}](tc.Arguments)
		if err != nil {
			return nil, err
		}

		return []string{args.Path}, err

	case moveFileToolName:

		args, err := tools.UnwrapArgs[struct {
			Src string `json:"src"`
			Dst string `json:"dst"`
		}](tc.Arguments)
		if err != nil {
			return nil, err
		}

		return []string{args.Src, args.Dst}, nil

	default:
		return nil, errIsNotFileTool
	}
}
