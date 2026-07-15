package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/prompt"
	"arch-agent/internal/tools"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
)

type FileSystemToolServer struct {
	*tools.BuildInToolServer
	fs *files.FileSystem
}

func NewFileSystemToolServer(fs *files.FileSystem) *FileSystemToolServer {
	return &FileSystemToolServer{
		fs: fs,
		BuildInToolServer: tools.NewBuildInToolServer(
			&DeleteTool{fs: fs},
			&EditFileTool{fs: fs},
			&MoveFileTool{fs: fs},
			&ListDirTool{fs: fs},
			&ReadFileTool{fs: fs},
			&SearchFilesTool{fs: fs},
			&WriteFileTool{fs: fs},
		),
	}
}

func NewRawFileSystemToolServer(fs *files.FileSystem) *tools.BuildInToolServer {
	return tools.NewBuildInToolServer(
		&DeleteTool{fs: fs},
		&EditFileTool{fs: fs},
		&MoveFileTool{fs: fs},
		&ListDirTool{fs: fs},
		&ReadFileTool{fs: fs},
		&SearchFilesTool{fs: fs},
		&WriteFileTool{fs: fs},
	)
}

func (r *FileSystemToolServer) AgentInstruction(agt agent.Agent) string {
	return prompt.FileSystemInstruction(r.fs.Cwd(), agt.ID(), agt.HasMemory())
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

	if e.IsDir() {
		return fmt.Sprintf("%s [directory]", label)
	}

	size := files.FormatSize(int(info.Size()))

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

func IsTextExt(p string) bool {
	switch strings.ToLower(path.Ext(p)) {
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

func IsImageExt(p string) bool {
	switch strings.ToLower(path.Ext(p)) {
	case "jpg", "jpeg", "png", "webp", "bmp":
		return true
	}
	return false
}
