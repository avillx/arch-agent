package files

import (
	"encoding/json"
	"sync"
)

type LLMFiles struct {
	fs *FileSystem
	mu sync.RWMutex
}

func NewLLMFiles(fs *FileSystem) *LLMFiles {
	return &LLMFiles{fs: fs}
}

func (s *LLMFiles) Read(id string, v any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.fs.ReadFile(id + ".json")
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func (s *LLMFiles) Write(id string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return s.fs.WriteToFile(id+".json", data)
}

func (s *LLMFiles) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.fs.DeleteFile(id + ".json")
}

func (s *LLMFiles) ListIDs() ([]string, error) {
	names, err := s.fs.ReadDir()
	if err != nil {
		return nil, nil
	}

	ids := make([]string, 0, len(names))
	for _, n := range names {
		if len(n) > 5 && n[len(n)-5:] == ".json" {
			ids = append(ids, n[:len(n)-5])
		}
	}
	return ids, nil
}
