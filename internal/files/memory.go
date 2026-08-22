package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"
	"strings"
)

var _ agent.MemoryIndexer = (*MemoryFiles)(nil)

type MemoryFiles struct {
	fs     *FileSystem
	logger *slog.Logger
}

func NewMemoryFiles(
	fs *FileSystem,
	logger *slog.Logger,
) *MemoryFiles {
	return &MemoryFiles{
		fs:     fs,
		logger: logger.WithGroup("memory files"),
	}
}

func (f *MemoryFiles) MemoryIndex(agentID agent.ID) (map[string]string, error) {

	memoryPath := resolveMemoryPath(agentID)
	index := map[string]string{}

	f.fs.WalkDir(memoryPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			f.logger.Error("walking dir", "path", p, "error", err)
			return nil
		}

		if d == nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		data, err := f.fs.ReadFile(p)
		if err != nil {
			f.logger.Error("read file", "path", p, "error", err)
			return nil
		}

		hook, err := resolveFrontmatter[struct {
			Hook string `yaml:"hook"`
		}](data)
		if err != nil {
			f.logger.Error("resolve frontmatter", "path", p, "error", err)
			return nil
		}

		index[path.Join(p)] = hook.Hook

		return nil
	})

	return index, nil
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
