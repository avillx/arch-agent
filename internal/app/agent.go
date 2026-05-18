package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/types"
	"context"
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

func (s *AgentService) getAgent(id agent.ID) (*agent.Agent, error) {
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

// TODO think about eliminating this func
func (s *AgentService) Chat(
	ctx context.Context,
	agentID agent.ID,
	additionalSysmtemPrompt string,
	preContextMessages []types.Message,
	contextMessages []types.Message,
	postContextMessages []types.Message,
	onResult func(result *agent.ReasonResult),
) (newMsgs []types.Message, err error) {

	a, err := s.getAgent(agentID)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, AgentContextKey, a)

	a.OnResult(onResult)

	conversation := []types.Message{}
	if preContextMessages != nil {
		conversation = append(conversation, preContextMessages...)
	}

	if contextMessages != nil {
		conversation = append(conversation, contextMessages...)
	}

	if postContextMessages != nil {
		conversation = append(conversation, postContextMessages...)
	}

	return a.Chat(ctx, additionalSysmtemPrompt, conversation)
}
