package files

import (
	"arch-agent/internal/agent"
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config
type AgentConfig struct {
	id           agent.ID
	description  string
	systemPrompt string
	reasoner     agent.LLMID
	tools        []string
}

func (c *AgentConfig) ID() agent.ID          { return c.id }
func (c *AgentConfig) Description() string   { return c.description }
func (c *AgentConfig) SystemPrompt() string  { return c.systemPrompt }
func (c *AgentConfig) Reasoner() agent.LLMID { return c.reasoner }
func (c *AgentConfig) Tools() []string       { return c.tools }

// Files
type AgentFiles struct {
	fs *FileSystem
	mu sync.RWMutex
}

func NewAgentFiles(fs *FileSystem) *AgentFiles {
	return &AgentFiles{fs: fs}
}

func (s *AgentFiles) Configs() ([]agent.AgentConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.fs.Dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var configs []agent.AgentConfig
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := agent.ID(strings.TrimPrefix(e.Name(), "agent."))
		cfg, err := s.readConfig(id)
		if err != nil {
			continue
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

func (s *AgentFiles) Config(id agent.ID) (agent.AgentConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readConfig(id)
}

func (s *AgentFiles) Save(cfg agent.AgentConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := marshalAgentFile(cfg)
	if err != nil {
		return err
	}
	return s.fs.WriteToFile(fmt.Sprintf("/agent.%s/agent.md", cfg.ID()), data)
}

// func (s *AgentFiles) Delete(id agent.ID) error {
// 	s.fs.D
// }

func (s *AgentFiles) readConfig(id agent.ID) (*AgentConfig, error) {
	data, err := s.fs.ReadFile(fmt.Sprintf("/agent.%s/agent.md", id))
	if err != nil {
		return nil, err
	}

	dto, systemPrompt, err := parseAgentFile(data)
	if err != nil {
		return nil, err
	}

	return &AgentConfig{
		id:           dto.ID,
		description:  dto.Description,
		systemPrompt: systemPrompt,
		reasoner:     dto.Reasoner,
		tools:        dto.Tools,
	}, nil
}

// DTO
type AgentConfigDTO struct {
	ID          agent.ID    `yaml:"id"`
	Description string      `yaml:"description,omitempty"`
	Reasoner    agent.LLMID `yaml:"reasoner"`
	Tools       []string    `yaml:"tools,omitempty"`
}

func parseAgentFile(data []byte) (AgentConfigDTO, string, error) {
	const delim = "---"
	s := strings.ReplaceAll(string(data), "\r\n", "\n")

	after, ok := strings.CutPrefix(s, delim+"\n")
	if !ok {
		return AgentConfigDTO{}, "", fmt.Errorf("agent file must start with ---")
	}

	fmEnd := strings.Index(after, "\n"+delim)
	if fmEnd == -1 {
		return AgentConfigDTO{}, "", fmt.Errorf("unclosed frontmatter")
	}

	var dto AgentConfigDTO
	if err := yaml.Unmarshal([]byte(after[:fmEnd]), &dto); err != nil {
		return AgentConfigDTO{}, "", err
	}

	systemPrompt := strings.TrimPrefix(after[fmEnd+len("\n"+delim):], "\n")
	return dto, systemPrompt, nil
}

func marshalAgentFile(cfg agent.AgentConfig) ([]byte, error) {
	fm, err := yaml.Marshal(AgentConfigDTO{
		ID:          cfg.ID(),
		Description: cfg.Description(),
		Reasoner:    cfg.Reasoner(),
		Tools:       cfg.Tools(),
	})
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fm)
	buf.WriteString("---\n")
	buf.WriteString(cfg.SystemPrompt())
	return buf.Bytes(), nil
}
