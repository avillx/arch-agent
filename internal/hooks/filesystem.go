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
	"strings"
)

const unixSeparator = "/"

var searchFilesToolName = (&fstools.SearchFilesTool{}).Name()
var readFileToolName = (&fstools.ReadFileTool{}).Name()   // read_file
var editFileToolName = (&fstools.EditFileTool{}).Name()   // edit_file
var deleteToolName = (&fstools.DeleteTool{}).Name()       // delete
var moveFileToolName = (&fstools.MoveFileTool{}).Name()   // move_file
var writeFileToolName = (&fstools.WriteFileTool{}).Name() // write_file
var listDirToolName = (&fstools.ListDirTool{}).Name()     // list_dir
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

func NewFileAccessHook(cwd string, rules ...Rule) (*FileAccessHook, error) {

	// validate patterns
	for _, r := range rules {
		if strings.Contains(r.Pattern, "**") {
			return nil, fmt.Errorf("rule '%s' :'**' is not supported", r.Pattern)
		}

		if _, err := path.Match(r.Pattern, ""); err != nil {
			return nil, err
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
	if !path.IsAbs(p) {
		p = path.Join(h.cwd, p)
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
		if match := matchPattern(r.Pattern, p); match {
			access = r.Access

			// rules sorted most-specific-first — first match wins
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

func matchPattern(pattern, p string) bool {
	if dir, ok := strings.CutSuffix(pattern, unixSeparator+"*"); ok {
		return p == dir || strings.HasPrefix(p, dir+unixSeparator)
	}
	match, _ := filepath.Match(pattern, p)
	return match
}

func isAllow(toolName agent.ToolName, a Access) bool {
	switch toolName {
	case searchFilesToolName, readFileToolName, listDirToolName:
		return a > No
	case editFileToolName, deleteToolName, writeFileToolName, moveFileToolName:
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
		deleteToolName,
		writeFileToolName,
		listDirToolName:

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

	case searchFilesToolName:
		args, err := tools.UnwrapArgs[struct {
			Root string `json:"root"`
		}](tc.Arguments)
		if err != nil {
			return nil, err
		}

		return []string{args.Root}, err

	default:
		return nil, errIsNotFileTool
	}
}

var _ runtime.ToolCallHook = (*OnlySupportedExtensionsHook)(nil)

type OnlySupportedExtensionsHook struct{}

func (h *OnlySupportedExtensionsHook) Apply(
	ctx context.Context,
	tc *agent.ToolCall,
) (*agent.ToolCall, error) {

	paths, err := resolvePaths(tc)
	if errors.Is(err, errIsNotFileTool) ||
		tc.ToolName == searchFilesToolName ||
		tc.ToolName == listDirToolName ||
		tc.ToolName == moveFileToolName ||
		tc.ToolName == deleteToolName {

		return tc, nil
	}

	// move for rename to supported case is allowed
	// couse is soft harness, fast check for caution agent
	// whenever agent should do this, it do this

	for _, p := range paths {
		if fstools.IsTextExt(p) || fstools.IsImageExt(p) {
			continue
		}

		errMsg := fmt.Sprintf(
			"You can read only text files and 'jpg,jpeg,png,webp,bmp', has: %s",
			path.Ext(p),
		)

		return nil, types.NewAgentMistakeError(errMsg)
	}

	return tc, nil
}

var _ runtime.CompletionHook = (*OnlyValidMemoryFrontmatterHook)(nil)

type OnlyValidMemoryFrontmatterHook struct {
	agentID agent.ID
	indexer agent.MemoryIndexer
}

func (h *OnlyValidMemoryFrontmatterHook) Apply(
	ctx context.Context,
	c *agent.Completion,
) (*agent.Completion, error) {

	if !c.Done {
		return c, nil
	}

	if _, err := h.indexer.MemoryIndex(h.agentID); err != nil {
		if joinedErrs, ok := err.(interface{ Unwrap() []error }); ok {
			var sb strings.Builder
			for _, e := range joinedErrs.Unwrap() {
				sb.WriteString(e.Error())
			}
			c.Done = false
			return c, types.NewAgentMistakeError(sb.String())
		}
	}

	return c, nil
}
