package knowledgefiles

import (
	"arch-agent/internal/app/knowledge"
	"arch-agent/internal/infra/storage/filesystem"
	"encoding/json"
	"os"
	"sync"
)

const IndexFile = "knowledge.json"

type Storage struct {
	filesystem filesystem.FileSystem
	mu         sync.RWMutex
}

func New(dir string) (*Storage, error) {
	fs, err := filesystem.New(dir)
	if err != nil {
		return nil, err
	}
	return &Storage{
		filesystem: fs,
	}, nil
}

func (f *Storage) Read(name string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := f.filesystem.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (f *Storage) AddNew(name, content string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.filesystem.WriteToFile(name, []byte(content))
}

func (f *Storage) Delete(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.filesystem.DeleteFile(name)
}

func (f *Storage) Override(name, content string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.filesystem.WriteToFile(name, []byte(content))
}

func (f *Storage) LoadIndex() (*knowledge.KnowledgeIndex, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := f.filesystem.ReadFile(IndexFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &knowledge.KnowledgeIndex{}, nil
		}
		return nil, err
	}

	var index knowledge.KnowledgeIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}

	return &index, nil
}

func (f *Storage) SaveIndex(idx *knowledge.KnowledgeIndex) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := json.MarshalIndent(idx, "", "	")
	if err != nil {
		return err
	}
	return f.filesystem.WriteToFile(IndexFile, data)
}
