package service

import (
	"arch-agent/internal/domain/agent"
	"fmt"
)

type AgentConfig struct {
	ID           agent.ID
	Description  string
	SystemPrompt string
	reasoner     LLMID
	toolServers  []string
}

type AgentConfigRepo interface {
	Configs() ([]AgentConfig, error)
	Config(agent.ID) (AgentConfig, error)
	Save(AgentConfig) error
	Delete(agent.ID) error
}

type LLMID string

type LLM interface {
	ID() LLMID
	Settings() any
	SetSettings(any) error
	agent.Reasoner
}

type LLMRepo interface {
	List() ([]LLMID, error)
	Get(LLMID) (LLM, error)
	Save(LLM) error
	Delete(LLMID) error
}

// Agent service
type AgentService struct {
	agentRepo   AgentConfigRepo
	llmRepo     LLMRepo
	toolService *ToolService
}

func NewAgentService(
	agentRepo AgentConfigRepo,
	llmRepo LLMRepo,
	toolService *ToolService,
) *AgentService {
	return &AgentService{
		agentRepo:   agentRepo,
		llmRepo:     llmRepo,
		toolService: toolService,
	}
}

func (s *AgentService) List() ([]AgentConfig, error) {
	return s.agentRepo.Configs()
}

func (s *AgentService) DeleteAgent(id agent.ID) error {
	return s.agentRepo.Delete(id)
}

func (s *AgentService) SaveAgent(cfg AgentConfig) error {

	// validate llm
	if _, err := s.llmRepo.Get(cfg.reasoner); err != nil {
		return err
	}

	// validate ToolService
	toolServersMap := map[string]struct{}{}
	for _, server := range s.toolService.Servers() {
		toolServersMap[server.Name()] = struct{}{}
	}

	for _, serverID := range cfg.toolServers {
		if _, ok := toolServersMap[serverID]; !ok {
			return fmt.Errorf("server %s is not exist", serverID)
		}
	}

	// save
	return s.agentRepo.Save(cfg)
}

func (s *AgentService) GetAgent(id agent.ID) (*agent.Agent, error) {
	config, err := s.agentRepo.Config(id)
	if err != nil {
		return nil, err
	}

	llm, err := s.llmRepo.Get(config.reasoner)
	if err != nil {
		return nil, err
	}

	toolKit := s.toolService.ToolKit(config.ID, config.toolServers)

	return agent.NewAgent(
		config.ID,
		config.Description,
		config.SystemPrompt,
		llm,
		toolKit,
	), nil
}
