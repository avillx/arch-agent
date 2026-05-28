package files

import (
	"arch-agent/internal/agent"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var _ agent.Repo = (*AgentFiles)(nil)

// Files
type AgentFiles struct {
	fs *FileSystem
	mu sync.RWMutex

	toolRegestry agent.ToolRegistry
}

func NewAgentFiles(
	fs *FileSystem,
	toolRegestry agent.ToolRegistry,
) *AgentFiles {
	return &AgentFiles{
		fs:           fs,
		toolRegestry: toolRegestry,
	}
}

func (s *AgentFiles) All() ([]agent.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.fs.Dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var dtos []AgentDTO
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		id := agent.ID(strings.TrimPrefix(e.Name(), "agent."))
		dto, err := s.readConfig(id)
		if err != nil {
			continue
		}
		dtos = append(dtos, dto)
	}

	agts, err := s.fromDTO(dtos...)
	if err != nil {
		return nil, err
	}

	return agts, nil
}

func (s *AgentFiles) Get(id agent.ID) (agent.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dto, err := s.readConfig(id)
	if err != nil {
		return nil, err
	}

	agts, err := s.fromDTO(dto)
	if err != nil {
		return nil, err
	}

	if len(agts) > 0 {
		return agts[0], nil
	}

	return nil, errors.New("can't get agent")
}

func (s *AgentFiles) Save(agt agent.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := marshalAgentFile(agt)
	if err != nil {
		return err
	}

	return s.fs.WriteToFile(fmt.Sprintf("/agent.%s/agent.md", agt.ID()), data)
}

// func (s *AgentFiles) Delete(id agent.ID) error {
// 	s.fs.D
// }

func (s *AgentFiles) readConfig(id agent.ID) (AgentDTO, error) {
	data, err := s.fs.ReadFile(fmt.Sprintf("/agent.%s/agent.md", id))
	if err != nil {
		return AgentDTO{}, err
	}

	return parseAgentFile(data)
}

func (s *AgentFiles) fromDTO(dtos ...AgentDTO) ([]agent.Agent, error) {
	agents := []agent.Agent{}

	for _, dto := range dtos {
		t, err := s.toolRegestry.GetTools(dto.Tools)
		if err != nil {
			return nil, err
		}

		agents = append(agents, agent.NewAgent(
			dto.ID,
			dto.Description,
			dto.SystemPrompt,
			dto.Model,
			t,
		))
	}

	return agents, nil
}

// DTO
type AgentDTO struct {
	ID           agent.ID      `yaml:"id"`
	Description  string        `yaml:"description,omitempty"`
	Model        agent.ModelID `yaml:"model"`
	SystemPrompt string        `yaml:"omitempty"`
	Tools        []string      `yaml:"tools,omitempty"`
}

func parseAgentFile(data []byte) (AgentDTO, error) {
	const delim = "---"
	s := strings.ReplaceAll(string(data), "\r\n", "\n")

	after, ok := strings.CutPrefix(s, delim+"\n")
	if !ok {
		return AgentDTO{}, fmt.Errorf("agent file must start with ---")
	}

	fmEnd := strings.Index(after, "\n"+delim)
	if fmEnd == -1 {
		return AgentDTO{}, fmt.Errorf("unclosed frontmatter")
	}

	var dto AgentDTO
	if err := yaml.Unmarshal([]byte(after[:fmEnd]), &dto); err != nil {
		return AgentDTO{}, err
	}

	dto.SystemPrompt = strings.TrimPrefix(after[fmEnd+len("\n"+delim):], "\n")
	return dto, nil
}

func marshalAgentFile(agt agent.Agent) ([]byte, error) {

	agentTools := agt.Tools()
	toolList := make([]string, len(agentTools))
	for i, t := range agentTools {
		toolList[i] = t.Name()
	}

	fm, err := yaml.Marshal(AgentDTO{
		ID:          agt.ID(),
		Description: agt.Description(),
		Model:       agt.Model(),
		Tools:       toolList,
	})
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fm)
	buf.WriteString("---\n")
	buf.WriteString(agt.SystemPrompt())
	return buf.Bytes(), nil
}
