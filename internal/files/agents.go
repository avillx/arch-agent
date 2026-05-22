package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type AgentFiles struct {
	fs *FileSystem
	mu sync.RWMutex
}

func NewAgentFiles(fs *FileSystem) *AgentFiles {
	return &AgentFiles{fs: fs}
}

func (s *AgentFiles) Configs() ([]chat.AgentConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.fs.Dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	configs := make([]chat.AgentConfig, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cfg, err := s.readConfig(agent.ID(e.Name()))
		if err != nil {
			continue
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

func (s *AgentFiles) Config(id agent.ID) (chat.AgentConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readConfig(id)
}

func (s *AgentFiles) Save(cfg chat.AgentConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "	")
	if err != nil {
		return err
	}
	if err := s.fs.WriteToFile(fmt.Sprintf("/agent.%s/agent.json", cfg.ID), data); err != nil {
		return err
	}

	return s.fs.WriteToFile(fmt.Sprintf("/agent.%s/agent.md", cfg.ID), []byte(cfg.SystemPrompt))
}

func (s *AgentFiles) Delete(id agent.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return os.RemoveAll(filepath.Join(s.fs.Dir(), string(id)))
}

func (s *AgentFiles) readConfig(id agent.ID) (chat.AgentConfig, error) {
	data, err := s.fs.ReadFile(fmt.Sprintf("/agent.%s/agent.json", id))
	if err != nil {
		return chat.AgentConfig{}, err
	}

	var cfg chat.AgentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return chat.AgentConfig{}, err
	}

	prompt, err := s.fs.ReadFile(fmt.Sprintf("/agent.%s/agent.md", id))
	if err == nil {
		cfg.SystemPrompt = string(prompt)
	}

	return cfg, nil
}
