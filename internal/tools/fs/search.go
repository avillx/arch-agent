package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/tools"
	"context"
	"fmt"
	"os"
	"path"
	"strings"
)

type SearchFilesTool struct {
	fs *files.FileSystem
}

func NewSearchFilesTool(fs *files.FileSystem) *SearchFilesTool {
	return &SearchFilesTool{fs: fs}
}

func (t *SearchFilesTool) Instruction() string {
	return `Search strategy:
- Use short, specific terms — function names, identifiers, keywords.
- Avoid natural language phrases; prefer exact tokens that appear in code or text.
- Narrow root to the smallest relevant directory to reduce noise.`
}

func (t *SearchFilesTool) Name() agent.ToolName { return "search_files" }
func (t *SearchFilesTool) Description() string {
	return "recursively search file contents under root; returns matching lines as path:line: text"
}
func (t *SearchFilesTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "root",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Directory to search under, e.g. './{your_folder}/memory/'",
		},
		{
			Name:        "query",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Case-insensitive substring to search for",
		},
		{
			Name:        "max_results",
			Required:    false,
			Type:        agent.TypeNumber,
			Description: "Maximum matches to return (Default: 20)",
		},
	}
}

func (t *SearchFilesTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Root       string `json:"root"`
		Query      string `json:"query"`
		MaxResults *int   `json:"max_results,omitempty"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	limit := 20
	if args.MaxResults != nil && *args.MaxResults > 0 {
		limit = *args.MaxResults
	}

	matches := collect(t.fs, args.Root, args.Query, limit)
	if len(matches) == 0 {
		return tools.Result("no matches found"), nil
	}

	result := strings.Join(matches, "\n")
	if len(matches) == limit {
		result += fmt.Sprintf("\n[limited to %d results]", limit)
	}
	return tools.Result(result), nil
}

func collect(fs interface {
	ReadDir(path string) ([]os.DirEntry, error)
	ReadFile(path string) ([]byte, error)
}, dirPath, query string, remaining int) []string {
	if remaining <= 0 {
		return nil
	}

	entries, err := fs.ReadDir(dirPath)
	if err == nil {
		var results []string
		for _, e := range entries {
			child := path.Join(dirPath, e.Name())
			results = append(results, collect(fs, child, query, remaining-len(results))...)
		}
		return results
	}

	data, err := fs.ReadFile(dirPath)
	if err != nil {
		return nil
	}

	return matchLines(dirPath, string(data), query, remaining)
}
