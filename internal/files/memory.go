package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var _ agent.MemoryIndexer = (*MemoryFiles)(nil)

type MemoryFiles struct {
	fs *FileSystem
}

func NewMemoryFiles(fs *FileSystem) *MemoryFiles {
	return &MemoryFiles{fs: fs}
}

func (f *MemoryFiles) MemoryIndex(agentID agent.ID) (map[string]string, error) {

	memoryPath := resolveMemoryPath(agentID)

	enties, err := f.fs.ReadDir(memoryPath)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var errs []error
	index := map[string]string{}
	for _, e := range enties {
		if e.IsDir() {
			continue
		}

		currentMemoryPath := path.Join(memoryPath, e.Name())
		data, err := f.fs.ReadFile(currentMemoryPath)
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

		index[path.Join("./", currentMemoryPath)] = hook.Hook
	}

	return index, errors.Join(errs...)
}

func (f *MemoryFiles) GetMemory(agentID agent.ID, name string) (string, error) {
	memoryPath := resolveMemoryPath(agentID)

	enties, err := f.fs.ReadDir(memoryPath)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return "", nil
		}
		return "", err
	}

	for _, e := range enties {
		if strings.TrimSuffix(e.Name(), path.Ext(e.Name())) == name {
			data, err := f.fs.ReadFile(path.Join(memoryPath, e.Name()))
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}

	return "", fmt.Errorf("agent %s has no memory %s : %w", agentID, name, types.ErrIsNotExist)
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
