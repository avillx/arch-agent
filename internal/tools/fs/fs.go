package fstools

import (
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
	Delete(path string) error
	ReadDir(path string) ([]string, error)
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
