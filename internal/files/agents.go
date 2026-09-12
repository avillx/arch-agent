package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var _ agent.Repo = (*AgentFiles)(nil)

const AgentFile = "agent.md"

// Files
type AgentFiles struct {
	storage FileStorage
	mu      sync.RWMutex
}

func NewAgentFiles(
	storage FileStorage,
) *AgentFiles {
	return &AgentFiles{
		storage: storage,
	}
}

func (s *AgentFiles) All() ([]agent.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := fs.ReadDir(s.storage.FS(), ".")
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

		dto, err := s.readConfig(agent.ID(e.Name()))
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
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("agent %s: %w", id, types.ErrIsNotExist)
		}
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

const sharedFolder = "shared"

func validateAgentName(agentName string) error {
	reservedNames := []string{
		MCPConfigFile,
		ModelsConfigFile,
		TaskConfigFile,
		SecretsConfigFile,
		MemoryConfigFile,
		TMPDir,
		skillsFolder,
		sharedFolder,
	}

	if slices.Contains(reservedNames, agentName) {
		return fmt.Errorf("name %s is reserved", agentName)
	}

	return nil
}

func (s *AgentFiles) Save(agt agent.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validateAgentName(string(agt.ID())); err != nil {
		return err
	}

	data, err := marshalAgentFile(agt)
	if err != nil {
		return err
	}

	if err := s.storage.MkdirAll(string(agt.ID()), ModeDirPerm); err != nil {
		return err
	}

	return s.storage.WriteFile(resolveAgentFilePath(agt.ID()), data, ModeFilePerm)
}

func (s *AgentFiles) Delete(agentID agent.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.storage.RemoveAll(fmt.Sprintf("/%s", agentID))
}

func (s *AgentFiles) readConfig(id agent.ID) (AgentDTO, error) {
	data, err := s.storage.ReadFile(resolveAgentFilePath(id))
	if err != nil {
		return AgentDTO{}, err
	}

	return parseAgentFile(data)
}

func (s *AgentFiles) fromDTO(dtos ...AgentDTO) ([]agent.Agent, error) {
	agents := []agent.Agent{}

	for _, dto := range dtos {
		agents = append(agents, agent.NewAgent(
			dto.ID,
			dto.Description,
			dto.SystemPrompt,
			dto.Model,
			dto.ToolServers,
			dto.HasMemory,
		))
	}

	return agents, nil
}

// DTO
type AgentDTO struct {
	ID           agent.ID `yaml:"id"`
	Description  string   `yaml:"description,omitempty"`
	Model        string   `yaml:"model"`
	SystemPrompt string   `yaml:"-"`
	ToolServers  []string `yaml:"tool_servers,omitempty"`
	HasMemory    bool     `yaml:"memory,omitempty"`
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

	fm, err := yaml.Marshal(AgentDTO{
		ID:          agt.ID(),
		Description: agt.Description(),
		Model:       agt.Model(),
		HasMemory:   agt.HasMemory(),
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

func resolveAgentFilePath(agentID agent.ID) string {
	return filepath.Join(string(agentID), AgentFile)
}
