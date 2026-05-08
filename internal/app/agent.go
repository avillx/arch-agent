package service

import (
	"arch-agent/internal/domain/agent"
	"fmt"
)

type AgentConfig struct {
	ID           agent.ID `json:"id"`
	Description  string   `json:"description,omitempty"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	Reasoner     LLMID    `json:"reasoner"`
	ToolServers  []string `json:"tool_servers,omitempty"`
}

type AgentConfigRepo interface {
	Configs() ([]AgentConfig, error)
	Config(agent.ID) (AgentConfig, error)
	Save(AgentConfig) error
	Delete(agent.ID) error
}

// Agent service
type AgentService struct {
	agentRepo   AgentConfigRepo
	llmService  *LLMService
	toolService *ToolService
}

func NewAgentService(
	agentRepo AgentConfigRepo,
	llmService *LLMService,
	toolService *ToolService,
) *AgentService {
	return &AgentService{
		agentRepo:   agentRepo,
		llmService:  llmService,
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
	if _, err := s.llmService.GetLLM(cfg.Reasoner); err != nil {
		return err
	}

	// validate ToolService
	toolServersMap := map[string]struct{}{}
	for _, server := range s.toolService.Servers() {
		toolServersMap[server.Name()] = struct{}{}
	}

	for _, serverID := range cfg.ToolServers {
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

	llm, err := s.llmService.GetLLM(config.Reasoner)
	if err != nil {
		return nil, err
	}

	toolKit := s.toolService.ToolKit(config.ID, config.ToolServers)

	return agent.NewAgent(
		config.ID,
		config.Description,
		config.SystemPrompt,
		llm,
		toolKit,
	), nil
}
