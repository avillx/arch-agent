package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/types"
	"context"
)

type AgentConfig struct {
	ID           agent.ID `json:"id"`
	Description  string   `json:"description,omitempty"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	Reasoner     LLMID    `json:"reasoner"`
	Tools        []string `json:"tools,omitempty"`
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

	// // validate ToolService
	// toolServersMap := map[string]struct{}{}
	// for _, server := range s.toolService.Servers() {
	// 	toolServersMap[server.Name()] = struct{}{}
	// }

	// for _, serverID := range cfg.ToolServers {
	// 	if _, ok := toolServersMap[serverID]; !ok {
	// 		return fmt.Errorf("server %s is not exist", serverID)
	// 	}
	// }

	// save
	return s.agentRepo.Save(cfg)
}

// TODO think about eliminating this func
func (s *AgentService) Chat(
	ctx context.Context,
	agentID agent.ID,
	additionalSysmtemPrompt string,
	conversation []types.Message,
	onResult func(result *agent.ReasonResult),
	additionalTools []string,
) (newMsgs []types.Message, err error) {

	config, err := s.agentRepo.Config(agentID)
	if err != nil {
		return nil, err
	}

	llm, err := s.llmService.GetLLM(config.Reasoner)
	if err != nil {
		return nil, err
	}

	tools, err := s.toolService.GetTools(append(config.Tools, additionalTools...))
	if err != nil {
		return nil, err
	}

	a := agent.NewAgent(
		config.ID,
		config.Description,
		config.SystemPrompt,
		llm,
		tools,
	)

	return a.Chat(ctx, additionalSysmtemPrompt, onResult, conversation)
}
