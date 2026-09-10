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

const MemoryFolder = "memory"

var _ agent.MemoryIndexer = (*MemoryFiles)(nil)

type MemoryFiles struct {
	storage FileStorage
	logger  *slog.Logger
}

func NewMemoryFiles(
	storage FileStorage,
	logger *slog.Logger,
) *MemoryFiles {
	return &MemoryFiles{
		storage: storage,
		logger:  logger.WithGroup("memory_files"),
	}
}

func (f *MemoryFiles) MemoryIndex(agentID agent.ID) (map[string]string, error) {

	// NOTE: all fs funcs disallow backslashes so that the reason to use path
	// over filepath. Cause filepath on windows return path with backslashes
	// and this stuff never read directory
	memoryPath := path.Join(string(agentID), MemoryFolder)
	index := map[string]string{}

	fs.WalkDir(f.storage.FS(), memoryPath, func(p string, d fs.DirEntry, err error) error {
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

		data, err := f.storage.ReadFile(p)
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

		index[filepath.Join(p)] = hook.Hook

		return nil
	})

	return index, nil
}

func (f *MemoryFiles) GetMemory(agentID agent.ID, name string) (string, error) {

	// NOTE: path over filepath is required
	memoryPath := path.Join(string(agentID), MemoryFolder)

	enties, err := fs.ReadDir(f.storage.FS(), memoryPath)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return "", nil
		}
		return "", err
	}

	for _, e := range enties {
		if strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())) == name {
			data, err := f.storage.ReadFile(filepath.Join(memoryPath, e.Name()))
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}

	return "", fmt.Errorf("agent %s has no memory %s : %w", agentID, name, types.ErrIsNotExist)
}
