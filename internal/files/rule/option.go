package ruledfiles

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/types"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var ErrReadOnly = fmt.Errorf("file is read only")

func ContainPath(base, target string) bool {
	return base == target || (strings.HasPrefix(base, target+"/") || strings.HasPrefix(base, target+"\\"))
}

func WithAccessOnPath(guardedPath string, read, write, append, delete bool) Option {

	// forbid operation with path and subpathes
	accessValidator := func(operationName string) func(string) (string, error) {
		return func(p string) (string, error) {
			if ContainPath(p, guardedPath) {
				return "", fmt.Errorf("you have no access to %s %s", operationName, p)
			}
			return p, nil
		}
	}

	// forbid operation with path and subpathes
	hiddenValidator := func(p string) (string, error) {
		if ContainPath(p, guardedPath) {
			return "", types.ErrIsNotExist
		}
		return p, nil
	}

	// hide path from ReadDir output
	dirOutputExcluder := func(dr DirResult) (DirResult, error) {

		// do not touch if not guarded
		dir := path.Dir(guardedPath)
		if !ContainPath(dr.path, dir) {
			return dr, nil
		}

		// get element that was hidden
		hiddenElement := path.Base(guardedPath)

		// exclude hidden element from read results
		dr.entries = slices.DeleteFunc(dr.entries, func(e os.DirEntry) bool {
			return e.Name() == hiddenElement
		})

		return dr, nil
	}

	return func(rfs *RuledFileSystem) {
		if !read {
			rfs.readDirPath.AddRule(hiddenValidator)
			rfs.readPath.AddRule(hiddenValidator)
			rfs.dirOutput.AddRule(dirOutputExcluder)
		}
		if !append {
			rfs.appendPath.AddRule(accessValidator("append for"))
		}
		if !write {
			rfs.writePath.AddRule(accessValidator("write in"))
		}
		if !delete {
			rfs.deletePath.AddRule(accessValidator("delete"))
		}
	}
}

func WithTrimPrefix(prefix string) Option {

	return func(rfs *RuledFileSystem) {
		prefixTrimmer := func(p string) (string, error) {
			return strings.TrimPrefix(p, prefix), nil
		}
		rfs.readDirPath.AddRule(prefixTrimmer)
		rfs.readPath.AddRule(prefixTrimmer)
		rfs.deletePath.AddRule(prefixTrimmer)
		rfs.appendPath.AddRule(prefixTrimmer)
		rfs.writePath.AddRule(prefixTrimmer)
	}
}

func WithPrefix(prefix string) Option {
	return func(rfs *RuledFileSystem) {
		addPrefix := func(p string) (string, error) {
			return path.Join(prefix, p), nil
		}

		rfs.readDirPath.AddRule(addPrefix)
		rfs.readPath.AddRule(addPrefix)
		rfs.deletePath.AddRule(addPrefix)
		rfs.appendPath.AddRule(addPrefix)
		rfs.writePath.AddRule(addPrefix)
	}
}

func WithCleanPathOnly() Option {
	return func(rfs *RuledFileSystem) {
		cleaner := func(p string) (string, error) {
			cleaned := path.Clean(p)
			if cleaned != p && cleaned+"/" != p {
				return "", fmt.Errorf("only clean paths allowed. avoid '/..' and '.' (current directory)")
			}
			return p, nil
		}

		rfs.readDirPath.AddRule(cleaner)
		rfs.readPath.AddRule(cleaner)
		rfs.deletePath.AddRule(cleaner)
		rfs.appendPath.AddRule(cleaner)
		rfs.writePath.AddRule(cleaner)
	}
}

func WithMount(mountPoint, targetPoint string) Option {
	return func(rfs *RuledFileSystem) {
		operationDispatcher := func(p string) (string, error) {
			if ContainPath(p, targetPoint) {
				return path.Join(mountPoint, strings.TrimPrefix(p, targetPoint)), nil
			}
			return p, nil
		}

		rfs.readDirPath.AddRule(operationDispatcher)
		rfs.readPath.AddRule(operationDispatcher)
		rfs.deletePath.AddRule(operationDispatcher)
		rfs.appendPath.AddRule(operationDispatcher)
		rfs.writePath.AddRule(operationDispatcher)

		directoryDispatcher := func(dirResult DirResult) (DirResult, error) {
			if dirResult.path != path.Dir(targetPoint) {
				return dirResult, nil
			}

			pathBase := path.Base(targetPoint)
			entries, err := rfs.fs.ReadDir(path.Dir(mountPoint))
			if err != nil {
				slog.Error("can't read mountpoint root", "mount point", dirResult.path, "error", err)
				return dirResult, nil
			}

			for _, e := range entries {
				if e.Name() == pathBase {
					dirResult.entries = append(dirResult.entries, e)
				}
			}

			return dirResult, nil
		}

		rfs.dirOutput.AddRule(directoryDispatcher)
	}
}

func WithNonEmptyRoot() Option {
	return func(rfs *RuledFileSystem) {
		normalizer := func(p string) (string, error) {
			if p == "" {
				return "/", nil
			}
			return p, nil
		}

		rfs.readDirPath.AddRule(normalizer)
	}
}

func WithReadOnlyTextFiles() Option {
	return func(rfs *RuledFileSystem) {
		rfs.readPath.AddRule(func(p string) (string, error) {
			if !isTextExt(p) {
				return "", fmt.Errorf("unreadable file extension %s", path.Ext(p))
			}
			return p, nil
		})
	}
}

func WithReadSizeLimit(sizeLimit int) Option {
	return func(rfs *RuledFileSystem) {
		rfs.readOutput.AddRule(readSizeLimiter(sizeLimit))
	}
}

func WithWriteSizeLimit(sizeLimit int) Option {
	return func(rfs *RuledFileSystem) {
		rfs.writeInput.AddRule(writeSizeLimiter(sizeLimit))
		rfs.appendOutput.AddRule(writeSizeLimiter(sizeLimit))
	}
}

var frontmatterRE = regexp.MustCompile(`(?s)\A---\r?\n.*?\r?\n---(?:\r?\n|$)`)

func WithoutFrontMatter() Option {

	// TODO:
	// it has edge case bug, works on read lines that range is starts with "---"
	// and contain another "---".
	// is can be any of markdown files.
	// now it's unneccecary but later issue should be closed
	unfrontmatter := func(b []byte) ([]byte, error) {
		return frontmatterRE.ReplaceAll(b, []byte{}), nil
	}

	return func(rfs *RuledFileSystem) {
		rfs.readOutput.AddRule(unfrontmatter)
	}

}

func WithWhiteListVisibility(dir string, whitelist ...string) Option {

	whitelistMap := map[string]struct{}{}
	for _, e := range whitelist {
		whitelistMap[path.Join(dir, e)] = struct{}{}
	}

	// Truncate visible entries
	truncator := func(dr DirResult) (DirResult, error) {
		if dr.path != dir {
			return dr, nil
		}

		dr.entries = slices.DeleteFunc(dr.entries, func(e os.DirEntry) bool {
			_, ok := whitelistMap[path.Join(dir, e.Name())]
			return !ok
		})

		return dr, nil
	}

	// forbid unallowed interactions
	whitelistCheck := func(p string) (string, error) {
		if !ContainPath(p, dir) {
			return p, nil
		}

		// allow for reading a directory
		if p == dir {
			return p, nil
		}

		for k := range whitelistMap {
			if ContainPath(p, k) {
				return p, nil
			}
		}

		return "", os.ErrNotExist
	}

	return func(rfs *RuledFileSystem) {
		rfs.dirOutput.AddRule(truncator)
		rfs.readDirPath.AddRule(whitelistCheck)
		rfs.readPath.AddRule(whitelistCheck)
		rfs.deletePath.AddRule(whitelistCheck)
		rfs.appendPath.AddRule(whitelistCheck)
		rfs.writePath.AddRule(whitelistCheck)
	}
}

func AgentAccessRules(agt agent.Agent) []Option {
	prefix := fmt.Sprintf("/agents/%s", agt.ID())
	const _10kb = 1024 * 10

	// Order is important!
	opts := []Option{
		// base opts.
		WithReadOnlyTextFiles(),
		WithTrimPrefix("/mnt"),
		WithNonEmptyRoot(),
		WithCleanPathOnly(),
		WithoutFrontMatter(),
		WithPrefix(prefix),
		WithAccessOnPath(path.Join(prefix, "/sessions"), false, false, false, false),
		WithAccessOnPath(path.Join(prefix, "/agent.md"), false, false, false, false),

		// shared files
		// TODO: all agents has access to shared files. make it as settable.
		WithAccessOnPath(path.Join(prefix, "/shared"), true, true, true, true),
		WithMount("/shared", path.Join(prefix, "/shared")),

		// size limit
		WithReadSizeLimit(_10kb),
		WithWriteSizeLimit(_10kb),
	}

	// memory opts
	if agt.HasMemory() {
		memoOpts := []Option{
			WithAccessOnPath(path.Join(prefix, "/memory"), true, false, true, false),
			WithAccessOnPath(path.Join(prefix, "/activity"), true, false, false, false),
		}
		opts = slices.Concat(opts, memoOpts)
	}

	// skills opts
	if len(agt.Skills()) > 0 {
		skillsOpts := []Option{
			WithAccessOnPath(path.Join(prefix, "/skills"), true, false, false, false),
			WithMount("/skills", path.Join(prefix, "/skills")),
			WithWhiteListVisibility("/skills", skillIDToPaths(agt.Skills())...),
		}
		opts = slices.Concat(opts, skillsOpts)
	}

	return opts
}

///

func AgentMemoryAccessRules(agentID agent.ID) []Option {
	prefix := fmt.Sprintf("/agents/%s", agentID)
	const _10kb = 1024 * 10

	// Order is important!
	opts := []Option{
		// base opts.
		WithReadOnlyTextFiles(),
		WithTrimPrefix("/mnt"),
		WithNonEmptyRoot(),
		WithCleanPathOnly(),
		WithPrefix(prefix),

		WithAccessOnPath(path.Join(prefix, "/sessions"), false, false, false, false),
		WithAccessOnPath(path.Join(prefix, "/agent.md"), false, false, false, false),
		WithAccessOnPath(path.Join(prefix, "/memory"), true, true, true, true),
		WithAccessOnPath(path.Join(prefix, "/activity"), true, false, false, false),

		// size limit
		WithReadSizeLimit(_10kb),
		WithWriteSizeLimit(_10kb),
	}

	return opts
}

func skillIDToPaths(skillIDs []agent.SkillID) []string {
	skillNames := make([]string, len(skillIDs))
	for i, s := range skillIDs {
		skillNames[i] = string(s)
	}
	return skillNames
}

func isTextExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case
		// docs and markups
		".txt", ".md", ".mdx", ".rst", ".tex", ".asciidoc", ".adoc",
		".csv", ".tsv", ".log", ".org",

		// data and conf's
		".json", ".json5", ".jsonl", ".jsonc", ".xml", ".yaml", ".yml",
		".toml", ".ini", ".cfg", ".conf", ".config", ".env",
		".properties", ".plist", ".hcl", ".tf", ".tfvars",
		".editorconfig", ".gitignore", ".gitattributes", ".dockerignore",

		// Web
		".html", ".htm", ".xhtml", ".css", ".scss", ".sass", ".less",
		".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs",
		".vue", ".svelte", ".astro", ".handlebars", ".hbs", ".ejs", ".pug",

		// Backend langs
		".go", ".py", ".rb", ".php", ".java", ".kt", ".kts",
		".scala", ".groovy", ".cs", ".fs", ".vb",
		".c", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".hxx",
		".rs", ".swift", ".m", ".mm", ".zig",
		".ex", ".exs", ".erl", ".hrl", ".clj", ".cljs",
		".hs", ".lhs", ".ml", ".mli", ".fsi",
		".lua", ".r", ".jl", ".dart", ".d",

		// scripts and shell
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".psm1", ".bat", ".cmd",

		// request and schemes
		".sql", ".graphql", ".gql", ".proto", ".thrift", ".avsc",

		// infra and CI
		".dockerfile", ".vagrantfile", ".makefile",
		".gradle", ".cmake", ".bazel", ".bzl",

		// other
		".diff", ".patch", ".lock", ".sum", ".mod", ".csproj", ".sln":
		return true
	}

	return false
}

func writeSizeLimiter(sizeLimit int) func(writeOp) (writeOp, error) {
	return func(op writeOp) (writeOp, error) {
		currentSize := len(op.data)
		if currentSize > sizeLimit {
			return writeOp{}, newErrOversize(currentSize, sizeLimit)
		}
		return op, nil
	}
}

func readSizeLimiter(sizeLimit int) func([]byte) ([]byte, error) {
	return func(b []byte) ([]byte, error) {
		currentSize := len(b)
		if currentSize > sizeLimit {
			return nil, newErrOversize(currentSize, sizeLimit)
		}
		return b, nil
	}
}

func newErrOversize(current, limit int) error {
	return fmt.Errorf(
		"file must be under %s current is %s",
		files.FormatSize(limit),
		files.FormatSize(current),
	)
}
