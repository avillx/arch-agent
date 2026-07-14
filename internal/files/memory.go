package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var _ runtime.MemoryIndexer = (*MemoryFiles)(nil)

type MemoryFiles struct {
	fs *FileSystem
}

func NewMemoryFiles(fs *FileSystem) *MemoryFiles {
	return &MemoryFiles{fs: fs}
}

func (f *MemoryFiles) MemoryIndex(agentID agent.ID) (string, error) {

	memoryPath := resolveMemoryPath(agentID)

	enties, err := f.fs.ReadDir(memoryPath)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return "", nil
		}
		return "", err
	}

	var errs []error
	var sb strings.Builder
	for _, e := range enties {
		if e.IsDir() {
			continue
		}

		data, err := f.fs.ReadFile(path.Join(memoryPath, e.Name()))
		if err != nil {
			errs = append(errs, err)
			continue
		}

		hook, err := resolveFrontmatter[struct {
			Hook string `yaml:"hook"`
		}](data)
		if err != nil {
			errs = append(errs, err)
		}

		fileName := e.Name()
		noExtName := strings.TrimSuffix(fileName, path.Ext(fileName))
		fmt.Fprintf(&sb, "(%s)[%s] - %s\n", noExtName, path.Join("./", memoryPath, fileName), hook.Hook)
	}

	return sb.String(), errors.Join(errs...)
}

func resolveMemoryPath(agentID agent.ID) string {
	return filepath.Join(string(agentID), "memory")
}

func resolveFrontmatter[T any](data []byte) (T, error) {
	var zero T
	const delim = "---"
	s := strings.ReplaceAll(string(data), "\r\n", "\n")

	after, ok := strings.CutPrefix(s, delim+"\n")
	if !ok {
		return zero, fmt.Errorf("hook file must start with ---")
	}

	fmEnd := strings.Index(after, "\n"+delim)
	if fmEnd == -1 {
		return zero, fmt.Errorf("unclosed frontmatter")
	}

	var dto T
	if err := yaml.Unmarshal([]byte(after[:fmEnd]), &dto); err != nil {
		return zero, err
	}

	return dto, nil
}
