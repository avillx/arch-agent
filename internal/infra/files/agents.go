package files

import (
	service "arch-agent/internal/app"
	"arch-agent/internal/domain/agent"
	"encoding/json"
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

func (s *AgentFiles) Configs() ([]service.AgentConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.fs.Dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	configs := make([]service.AgentConfig, 0, len(entries))
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

func (s *AgentFiles) Config(id agent.ID) (service.AgentConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readConfig(id)
}

func (s *AgentFiles) Save(cfg service.AgentConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	agentFS := s.fs.Sub(string(cfg.ID))

	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := agentFS.WriteToFile("agent.json", data); err != nil {
		return err
	}

	return agentFS.WriteToFile("agent.md", []byte(cfg.SystemPrompt))
}

func (s *AgentFiles) Delete(id agent.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return os.RemoveAll(filepath.Join(s.fs.Dir(), string(id)))
}

func (s *AgentFiles) readConfig(id agent.ID) (service.AgentConfig, error) {
	agentFS := s.fs.Sub(string(id))

	data, err := agentFS.ReadFile("agent.json")
	if err != nil {
		return service.AgentConfig{}, err
	}

	var cfg service.AgentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return service.AgentConfig{}, err
	}

	prompt, err := agentFS.ReadFile("agent.md")
	if err == nil {
		cfg.SystemPrompt = string(prompt)
	}

	return cfg, nil
}

func (s *AgentFiles) Knowledges(id agent.ID) *KnowledgeFiles {
	return NewKnowledgeFiles(s.fs.Sub(filepath.Join(string(id), "knowledges")))
}

func (s *AgentFiles) Activity(id agent.ID) *ActivityFiles {
	return NewActivityFiles(s.fs.Sub(filepath.Join(string(id), "activity")))
}

func (s *AgentFiles) Sessions(id agent.ID) *SessionFiles {
	return NewSessionFiles(s.fs.Sub(filepath.Join(string(id), "sessions")))
}
