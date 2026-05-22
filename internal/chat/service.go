package chat

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/llm"
	"context"
)

type AgentConfig struct {
	ID           agent.ID  `json:"id"`
	Description  string    `json:"description,omitempty"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
	Reasoner     llm.LLMID `json:"reasoner"`
	Tools        []string  `json:"tools,omitempty"`
}

type AgentConfigRepo interface {
	Configs() ([]AgentConfig, error)
	Config(agent.ID) (AgentConfig, error)
	Save(AgentConfig) error
	Delete(agent.ID) error
}

// TODO: replace llm.Service with repo interface
// type ReasonerRepo interface {
// 	GetLLM(string) agent.Reasoner
// }

type ToolRegistry interface {
	GetTools([]string) ([]agent.Tool, error)
}

// Agent service
type Service struct {
	agentRepo    AgentConfigRepo
	llmService   *llm.Service
	toolRegistry ToolRegistry
}

func NewService(
	agentRepo AgentConfigRepo,
	llmService *llm.Service,
	toolRegistry ToolRegistry,
) *Service {
	return &Service{
		agentRepo:    agentRepo,
		llmService:   llmService,
		toolRegistry: toolRegistry,
	}
}

func (s *Service) List() ([]AgentConfig, error) {
	return s.agentRepo.Configs()
}

func (s *Service) DeleteAgent(id agent.ID) error {
	return s.agentRepo.Delete(id)
}

func (s *Service) SaveAgent(cfg AgentConfig) error {

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
func (s *Service) Chat(
	ctx context.Context,
	agentID agent.ID,
	additionalSysmtemPrompt string,
	conversation []agent.Message,
	onResult func(result *agent.ReasonResult),
	additionalTools []string,
) (newMsgs []agent.Message, err error) {

	config, err := s.agentRepo.Config(agentID)
	if err != nil {
		return nil, err
	}

	llm, err := s.llmService.GetLLM(config.Reasoner)
	if err != nil {
		return nil, err
	}

	tools, err := s.toolRegistry.GetTools(append(config.Tools, additionalTools...))
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
