package hooks

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"arch-agent/internal/tools"
	fstools "arch-agent/internal/tools/fs"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// actualHarnesses

// OnToolCall
// - (without access) forbrid read paths .env, mcp_servers.json, tasks.json, models.json, unprovidied skills
// - (with read only) forbrid edit paths sessions providied skills
// - (with append only) forbrid edit paths
// - (with edit only) forbrid append on paths

// OnToolCallResultMessage
// - tooBigFile
// - unsupprotedFormat
// - contain data:// image file
// - hide directories sessions, agent.md, .secrets from read dir
// - remove frontmatter from skills

// OnComplete
// - toolRepeat loop error
// - end + empty message
// - end + unendedTodos

// x := &fstools.EditFileTool{}
// x.Name()

// prefix := fmt.Sprintf("/%s", agt.ID())
// const _10kb = 1024 * 10

// Order is important!
// opts := []Option{
// // base opts.
// WithReadOnlyTextFiles(),
// WithTrimPrefix("/mnt"),
// WithNonEmptyRoot(),
// WithCleanPathOnly(),
// WithoutFrontMatter(),
// WithPrefix(prefix),
// WithAccessOnPath(path.Join(prefix, "/sessions"), false, false, false, false),
// WithAccessOnPath(path.Join(prefix, "/agent.md"), false, false, false, false),

// // shared files
// // TODO: all agents has access to shared files. make it as settable.
// WithAccessOnPath(path.Join(prefix, "/shared"), true, true, true, true),
// WithMount("/shared", path.Join(prefix, "/shared")),

// // size limit
// WithReadSizeLimit(_10kb),
// WithWriteSizeLimit(_10kb),
// }

// // memory opts
// if agt.HasMemory() {
// 	memoOpts := []Option{
// 		WithAccessOnPath(path.Join(prefix, "/memory"), true, false, true, false),
// 		WithAccessOnPath(path.Join(prefix, "/activity"), true, false, false, false),
// 	}
// 	opts = slices.Concat(opts, memoOpts)
// }

// // skills opts
// if len(agt.Skills()) > 0 {
// 	skillsOpts := []Option{
// 		WithAccessOnPath(path.Join(prefix, "/skills"), true, false, false, false),
// 		WithMount("/skills", path.Join(prefix, "/skills")),
// 		WithWhiteListVisibility("/skills", skillIDToPaths(agt.Skills())...),
// 	}
// 	opts = slices.Concat(opts, skillsOpts)
// }

////

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

type FileAccessHook struct {
	rulesFactory func(agent.Agent) []Rule
	cwd          string
}

func NewFileAccessHook(cwd string, rulesFactory func(agent.Agent) []Rule) (*FileAccessHook, error) {

	// validate patterns
	for _, r := range rulesFactory(agent.NewAgent("", "", "", "", nil, nil, false)) {
		if strings.Contains(r.Pattern, "**") {
			return nil, fmt.Errorf("rule '%s' :'**' is not supported", r.Pattern)
		}

		if _, err := path.Match(r.Pattern, ""); err != nil {
			return nil, err
		}
	}

	return &FileAccessHook{
		rulesFactory: rulesFactory,
		cwd:          cwd,
	}, nil
}

func (h *FileAccessHook) resolveRules(agt agent.Agent) []Rule {
	rules := h.rulesFactory(agt)
	for i, r := range rules {
		if !path.IsAbs(r.Pattern) {
			r.Pattern = path.Join(h.cwd, r.Pattern)
		}

		rules[i] = Rule{Pattern: r.Pattern, Access: r.Access}
	}

	// most specific (longest) pattern first
	sort.SliceStable(rules, func(i, j int) bool {
		return len(rules[i].Pattern) > len(rules[j].Pattern)
	})

	return rules
}

func (h *FileAccessHook) Apply(sessID session.ID, agt agent.Agent, tc *agent.ToolCall) (*agent.ToolCall, error) {

	paths, err := resolvePaths(tc)
	if err != nil {
		// if not file tool then just ignore
		if errors.Is(err, errIsNotFileTool) {
			return tc, nil
		}
		return nil, err
	}

	for _, p := range paths {
		if err := h.verifyPath(agt, tc.ToolName, path.Clean(p)); err != nil {
			return nil, err
		}
	}

	return tc, nil
}

func (h *FileAccessHook) verifyPath(agt agent.Agent, toolName agent.ToolName, p string) error {

	if !path.IsAbs(p) { // /skills/test_note.md is abs for this validate
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
	for _, r := range h.resolveRules(agt) {
		if match := matchPattern(r.Pattern, p); match {
			access = r.Access
			break // rules sorted most-specific-first — first match wins
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

const unixSeparator = "/"

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
