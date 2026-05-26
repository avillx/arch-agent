package agent

import (
	"context"
)

type ChatSvc interface {
	Chat(ctx context.Context,
		agentID ID,
		additionalSysmtemPrompt string,
		conversation []Message,
		onResult func(result *ReasonResult),
		additionalTools []string,
	) (newMsgs []Message, err error)
}

type AgentService interface {
	List() ([]AgentConfig, error)
	SaveAgent(cfg AgentConfig) error
}

type LLMID string

type AgentConfig interface {
	ID() ID
	Description() string
	SystemPrompt() string
	Reasoner() LLMID
	Tools() []string
}

type AgentConfigRepo interface {
	Configs() ([]AgentConfig, error)
	Config(ID) (AgentConfig, error)
	Save(AgentConfig) error
	// Delete(agent.ID) error
}

type ReasonerRegistry interface {
	GetLLM(LLMID) (Reasoner, error)
}

type ToolRegistry interface {
	GetTools([]string) ([]Tool, error)
}

type service struct {
	agentRepo    AgentConfigRepo
	reasonerReg  ReasonerRegistry
	toolRegistry ToolRegistry
}

func NewService(
	agentRepo AgentConfigRepo,
	reasonerReg ReasonerRegistry,
	toolRegistry ToolRegistry,
) *service {
	return &service{
		agentRepo:    agentRepo,
		reasonerReg:  reasonerReg,
		toolRegistry: toolRegistry,
	}
}

func (s *service) List() ([]AgentConfig, error) {
	return s.agentRepo.Configs()
}

// func (s *Service) DeleteAgent(id agent.ID) error {
// 	return s.agentRepo.Delete(id)
// }

func (s *service) SaveAgent(cfg AgentConfig) error {

	// validate llm
	// if _, err := s.llmService.GetLLM(cfg.Reasoner()); err != nil {
	// 	return err
	// }

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
func (s *service) Chat(
	ctx context.Context,
	agentID ID,
	additionalSysmtemPrompt string,
	conversation []Message,
	onResult func(result *ReasonResult),
	additionalTools []string,
) (newMsgs []Message, err error) {

	config, err := s.agentRepo.Config(agentID)
	if err != nil {
		return nil, err
	}

	llm, err := s.reasonerReg.GetLLM(config.Reasoner())
	if err != nil {
		return nil, err
	}

	tools, err := s.toolRegistry.GetTools(append(config.Tools(), additionalTools...))
	if err != nil {
		return nil, err
	}

	a := NewAgent(
		config.ID(),
		config.Description(),
		config.SystemPrompt(),
		llm,
		tools,
	)

	return a.Chat(ctx, additionalSysmtemPrompt, onResult, conversation)
}
