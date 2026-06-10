package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"fmt"
	"path"
	"strings"
)

// search_files
type SearchFilesTool struct{ fs FS }

func NewSearchFilesTool(fs FS) *SearchFilesTool { return &SearchFilesTool{fs} }

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
			Description: "Directory to search under, e.g. file:///some_dir/",
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

func (t *SearchFilesTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
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

	// TODO: add error checking. via "AS"
	entries, err := t.fs.ReadDir(internalPath)
	if err == nil { // this close edge case for pathToFile/pathToDir
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
