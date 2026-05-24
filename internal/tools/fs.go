package tools

import (
	"arch-agent/internal/agent"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type FS interface {
	ReadFile(path string) ([]byte, error)
	WriteToFile(path string, data []byte) error
	AppendToFile(path string, data []byte) error
	DeleteFile(path string) error
	ReadDir(path string) ([]string, error)
}

// list_dir
type ListDirTool struct{ fs FS }

func NewListDirTool(fs FS) *ListDirTool { return &ListDirTool{fs} }

func (t *ListDirTool) Name() string { return "list_dir" }

func (t *ListDirTool) Description() string {
	return "list entries in a directory; returns one file:/// path per line"
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
	args, err := unwrapArgs[struct {
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
		lines[i] = toAgent(path.Join(internal, name))
	}
	return strings.Join(lines, "\n"), nil
}

// read_file

type ReadFileTool struct{ fs FS }

func NewReadFileTool(fs FS) *ReadFileTool { return &ReadFileTool{fs} }

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Description() string {
	return "read file content, optionally limited to a line range (1-indexed)"
}
func (t *ReadFileTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "file path, e.g. file:///notes/README.md",
		},
		{
			Name:        "start_line",
			Required:    false,
			Type:        agent.TypeNumber,
			Description: "first line to read (default: 1)",
		},
		{
			Name:        "end_line",
			Required:    false,
			Type:        agent.TypeNumber,
			Description: "last line to read (default: end of file)",
		},
	}
}

func (t *ReadFileTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := unwrapArgs[struct {
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

	if start > 1 || end < total {
		return fmt.Sprintf("[lines %d–%d of %d]\n%s", start, end, total, content), nil
	}
	return content, nil
}

// write_file
type WriteFileTool struct{ fs FS }

func NewWriteFileTool(fs FS) *WriteFileTool { return &WriteFileTool{fs} }

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string {
	return `write content to a file, creating it if it does not exist.
mode: "overwrite" (default) replaces the file; "append" adds to the end`
}
func (t *WriteFileTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "file path, e.g. file:///notes/README.md",
		},
		{
			Name:        "content",
			Required:    true,
			Type:        agent.TypeString,
			Description: "text content to write",
		},
		{
			Name:        "mode",
			Required:    false,
			Type:        agent.TypeString,
			Description: `"overwrite" (default) or "append"`,
		},
	}
}

func (t *WriteFileTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := unwrapArgs[struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Mode    string `json:"mode"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	if IsReadOnly(args.Path) {
		return "", fmt.Errorf("this path is read only")
	}

	if !IsTextFile(args.Path) {
		return "", fmt.Errorf("you can write only files with text extensions")
	}

	internal, err := toInternal(args.Path)
	if err != nil {
		return "", err
	}

	data := []byte(args.Content)

	if args.Mode == "append" {
		err = t.fs.AppendToFile(internal, data)
	} else {
		err = t.fs.WriteToFile(internal, data)
	}
	if err != nil {
		return "", wrapFSError(err, args.Path)
	}

	return fmt.Sprintf("wrote %d bytes to %s", len(data), args.Path), nil
}

// edit_file
type EditFileTool struct{ fs FS }

func NewEditFileTool(fs FS) *EditFileTool { return &EditFileTool{fs} }

func (t *EditFileTool) Name() string { return "edit_file" }
func (t *EditFileTool) Description() string {
	return "replace a unique string in a file; old_str must match exactly once"
}

func (t *EditFileTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "file path",
		},
		{
			Name:        "old_str",
			Required:    true,
			Type:        agent.TypeString,
			Description: "the unique string to find",
		},
		{
			Name:        "new_str",
			Required:    true,
			Type:        agent.TypeString,
			Description: "replacement string (empty string to delete)",
		},
	}
}

func (t *EditFileTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := unwrapArgs[struct {
		Path   string `json:"path"`
		OldStr string `json:"old_str"`
		NewStr string `json:"new_str"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	if IsReadOnly(args.Path) {
		return "", fmt.Errorf("this path is read only")
	}

	if !IsTextFile(args.Path) {
		return "", fmt.Errorf("you can edit files only with text extensions")
	}

	internal, err := toInternal(args.Path)
	if err != nil {
		return "", err
	}

	data, err := t.fs.ReadFile(internal)
	if err != nil {
		return "", wrapFSError(err, args.Path)
	}

	content := string(data)
	count := strings.Count(content, args.OldStr)

	switch count {
	case 0:
		return "", fmt.Errorf("%s: old_str not found", args.Path)
	case 1:
		// exactly one match — safe to replace
	default:
		return "", fmt.Errorf("%s: old_str found %d times, must be unique", args.Path, count)
	}

	updated := strings.Replace(content, args.OldStr, args.NewStr, 1)
	if err := t.fs.WriteToFile(internal, []byte(updated)); err != nil {
		return "", wrapFSError(err, args.Path)
	}

	return fmt.Sprintf("edited %s", args.Path), nil
}

// move_file
type MoveFileTool struct{ fs FS }

func NewMoveFileTool(fs FS) *MoveFileTool { return &MoveFileTool{fs} }

func (t *MoveFileTool) Name() string { return "move_file" }
func (t *MoveFileTool) Description() string {
	return "move or rename a file from src to dst"
}
func (t *MoveFileTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "src",
			Required:    true,
			Type:        agent.TypeString,
			Description: "source file path",
		},
		{
			Name:        "dst",
			Required:    true,
			Type:        agent.TypeString,
			Description: "destination file path",
		},
	}
}

func (t *MoveFileTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := unwrapArgs[struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	if IsReadOnly(args.Src) {
		return "", fmt.Errorf("this path is read only")
	}

	if !IsTextFile(args.Src) && IsTextFile(args.Dst) {
		return "", fmt.Errorf("can't change non text file extension to text")
	}

	srcInternal, err := toInternal(args.Src)
	if err != nil {
		return "", err
	}
	dstInternal, err := toInternal(args.Dst)
	if err != nil {
		return "", err
	}

	data, err := t.fs.ReadFile(srcInternal)
	if err != nil {
		return "", wrapFSError(err, args.Src)
	}

	if err := t.fs.WriteToFile(dstInternal, data); err != nil {
		return "", wrapFSError(err, args.Dst)
	}

	if err := t.fs.DeleteFile(srcInternal); err != nil {
		// dst was written successfully; src still exists — inform the agent
		return "", fmt.Errorf("file copied to %s but %s", args.Dst, wrapFSError(err, args.Src))
	}

	return fmt.Sprintf("moved %s → %s", args.Src, args.Dst), nil
}

// delete_file
type DeleteFileTool struct{ fs FS }

func NewDeleteFileTool(fs FS) *DeleteFileTool { return &DeleteFileTool{fs} }

func (t *DeleteFileTool) Name() string { return "delete_file" }
func (t *DeleteFileTool) Description() string {
	return "permanently delete a file; this operation cannot be undone"
}
func (t *DeleteFileTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "path",
			Required:    true,
			Type:        agent.TypeString,
			Description: "file path to delete",
		},
	}
}

func (t *DeleteFileTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := unwrapArgs[struct {
		Path string `json:"path"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	if IsReadOnly(args.Path) {
		return "", fmt.Errorf("this path is read only")
	}

	internal, err := toInternal(args.Path)
	if err != nil {
		return "", err
	}

	if err := t.fs.DeleteFile(internal); err != nil {
		return "", wrapFSError(err, args.Path)
	}

	return fmt.Sprintf("deleted %s", args.Path), nil
}

// search_files

type SearchFilesTool struct{ fs FS }

func NewSearchFilesTool(fs FS) *SearchFilesTool { return &SearchFilesTool{fs} }

func (t *SearchFilesTool) Name() string { return "search_files" }
func (t *SearchFilesTool) Description() string {
	return "recursively search file contents under root; returns matching lines as path:line: text"
}
func (t *SearchFilesTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "root",
			Required:    true,
			Type:        agent.TypeString,
			Description: "directory to search under, e.g. file:///some_dir/",
		},
		{
			Name:        "query",
			Required:    true,
			Type:        agent.TypeString,
			Description: "case-insensitive substring to search for",
		},
		{
			Name:        "max_results",
			Required:    false,
			Type:        agent.TypeNumber,
			Description: "maximum matches to return (default: 20)",
		},
	}
}

func (t *SearchFilesTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := unwrapArgs[struct {
		Root       string `json:"root"`
		Query      string `json:"query"`
		MaxResults *int   `json:"max_results,omitempty"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	rootInternal, err := toInternal(args.Root)
	if err != nil {
		return "", err
	}

	limit := 20
	if args.MaxResults != nil && *args.MaxResults > 0 {
		limit = *args.MaxResults
	}

	matches := t.collect(rootInternal, args.Query, limit)
	if len(matches) == 0 {
		return "no matches found", nil
	}

	result := strings.Join(matches, "\n")
	if len(matches) == limit {
		result += fmt.Sprintf("\n[limited to %d results]", limit)
	}
	return result, nil
}

// collect recurses the tree by probing ReadDir first.
// If ReadDir fails the path is a file — probe ReadFile instead.
// Errors on individual nodes are silently skipped to keep search resilient.
func (t *SearchFilesTool) collect(internalPath, query string, remaining int) []string {
	if remaining <= 0 {
		return nil
	}

	entries, err := t.fs.ReadDir(internalPath)
	if err == nil { // this colose edge case for pathToFile/pathToDir
		var results []string
		for _, name := range entries {
			child := path.Join(internalPath, name)
			results = append(results, t.collect(child, query, remaining-len(results))...)
		}
		return results
	}

	data, err := t.fs.ReadFile(internalPath)
	if err != nil {
		return nil
	}

	return matchLines(toAgent(internalPath), string(data), query, remaining)
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

const fileScheme = "file:///"
const fileDirectory = "files/"

// toInternal strips the file:/// scheme and resolves ".." against the virtual
// root, so file:///../../etc/passwd safely becomes "etc/passwd".
func toInternal(agentPath string) (string, error) {
	raw, ok := strings.CutPrefix(agentPath, fileScheme)
	if !ok {
		return "", fmt.Errorf("path must start with %s, got: %q", fileScheme, agentPath)
	}
	cleaned := path.Clean("/" + raw) // resolves ".." against virtual root
	return fileDirectory + strings.TrimPrefix(cleaned, "/"), nil
}

func toAgent(internalPath string) string {
	return fileScheme + strings.TrimPrefix(internalPath, fileDirectory)
}

// wrapFSError replaces the real OS path in the error with the agent's virtual
// path. syscall.Errno (inside *os.PathError) carries no path — only a message
// like "no such file or directory" — so the real location stays hidden.
func wrapFSError(err error, agentPath string) error {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return fmt.Errorf("%s: %s", agentPath, pathErr.Err)
	}
	return fmt.Errorf("%s: operation failed", agentPath)
}

func IsReadOnly(path string) bool {
	switch {
	case strings.Contains(path, "file:///skills/"):
		return true
	case strings.Contains(path, "file:///activity/"):
		return true
	}
	return false
}

func IsTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case
		// Документы и разметка
		".txt", ".md", ".mdx", ".rst", ".tex", ".asciidoc", ".adoc",
		".csv", ".tsv", ".log", ".org",

		// Данные и конфиги
		".json", ".json5", ".jsonc", ".xml", ".yaml", ".yml",
		".toml", ".ini", ".cfg", ".conf", ".config", ".env",
		".properties", ".plist", ".hcl", ".tf", ".tfvars",
		".editorconfig", ".gitignore", ".gitattributes", ".dockerignore",

		// Web
		".html", ".htm", ".xhtml", ".css", ".scss", ".sass", ".less",
		".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs",
		".vue", ".svelte", ".astro", ".handlebars", ".hbs", ".ejs", ".pug",

		// Backend языки
		".go", ".py", ".rb", ".php", ".java", ".kt", ".kts",
		".scala", ".groovy", ".cs", ".fs", ".vb",
		".c", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".hxx",
		".rs", ".swift", ".m", ".mm", ".zig",
		".ex", ".exs", ".erl", ".hrl", ".clj", ".cljs",
		".hs", ".lhs", ".ml", ".mli", ".fsi",
		".lua", ".r", ".jl", ".dart", ".d",

		// Скрипты и шелл
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".psm1", ".bat", ".cmd",

		// Запросы и схемы
		".sql", ".graphql", ".gql", ".proto", ".thrift", ".avsc",

		// Инфраструктура и CI
		".dockerfile", ".vagrantfile", ".makefile",
		".gradle", ".cmake", ".bazel", ".bzl",

		// Прочее
		".diff", ".patch", ".lock", ".sum", ".mod", ".csproj", ".sln":
		return true
	}

	return false
}
