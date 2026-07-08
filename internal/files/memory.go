package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"fmt"
	"os"
	"path/filepath"
)

const MemIndexFile = "INDEX.md"

var _ runtime.MemoryIndexer = (*MemoryFiles)(nil)

type MemoryFiles struct {
	fs *FileSystem
}

func NewMemoryFiles(fs *FileSystem) *MemoryFiles {
	return &MemoryFiles{fs: fs}
}

func (f *MemoryFiles) GetMemoryIndex(agentID agent.ID) (string, error) {

	path := resolvePathToIndex(agentID)
	data, err := f.fs.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", f.fs.WriteToFile(path, []byte{})
		}

		return "", fmt.Errorf("read %s: %w", modelSettingsFile, err)
	}

	return string(data), nil
}

func resolvePathToIndex(agentID agent.ID) string {
	return filepath.Join(string(agentID), "memory", MemIndexFile)
}
