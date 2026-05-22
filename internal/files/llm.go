package files

import (
	"arch-agent/internal/llm"
	"encoding/json"
	"maps"
	"os"
	"sync"
)

const llmSettingsFile = "llms.json"

type LLMFiles struct {
	fs *FileSystem
	mu sync.RWMutex
}

func NewLLMFiles(fs *FileSystem) *LLMFiles {
	return &LLMFiles{fs: fs}
}

func (s *LLMFiles) Load() (map[llm.LLMID]llm.LLMSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.fs.ReadFile(llmSettingsFile)
	if err != nil {
		if os.IsNotExist(err) {
			if err := s.fs.touchFile("llms.json"); err != nil {
				return nil, err
			}
			return map[llm.LLMID]llm.LLMSettings{}, nil
		}
		return nil, err
	}

	var settings map[llm.LLMID]llm.LLMSettings

	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return settings, nil
}

func (s *LLMFiles) Save(id llm.LLMID, settingsUpdate llm.LLMSettings) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.Load()
	if err != nil {
		return err
	}

	maps.Insert(settings[id], maps.All(settingsUpdate))

	data, err := json.MarshalIndent(settings, "", "	")
	if err != nil {
		return err
	}

	return s.fs.WriteToFile(llmSettingsFile, data)
}

func (s *LLMFiles) Delete(id llm.LLMID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.Load()
	if err != nil {
		return err
	}

	delete(settings, id)

	data, err := json.MarshalIndent(settings, "", "	")
	if err != nil {
		return err
	}

	return s.fs.WriteToFile(llmSettingsFile, data)
}
