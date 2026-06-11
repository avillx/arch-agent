package chat

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"context"
)

type SessionSkillRepo interface {
	Get(session.ID) ([]agent.SkillID, error)
}

type SkillRegestry interface {
	Get(...agent.SkillID) ([]agent.Skill, error)
}

type Service struct {
	agentRepo    agent.Repo
	sessionSvc   *session.Service
	modelRepo    agent.ModelRepository
	toolRegistry agent.ToolRegistry
	// sessSkillRepo SessionSkillRepo
	// skillRegestry SkillRegestry
	runtime *runtime.AgentRuntime
}

func NewService(
	agentRepo agent.Repo,
	sessionSvc *session.Service,
	modelRepo agent.ModelRepository,
	toolRegistry agent.ToolRegistry,
	runtime *runtime.AgentRuntime,
) *Service {
	return &Service{
		agentRepo:    agentRepo,
		sessionSvc:   sessionSvc,
		modelRepo:    modelRepo,
		toolRegistry: toolRegistry,
		runtime:      runtime,
	}
}

func (s *Service) Chat(
	ctx context.Context,
	agentID agent.ID,
	sessionID session.ID,
	request *agent.UserMessage,
	reader runtime.EventReader,
	providedTools []agent.Tool,
	logging bool,
) error {

	// session
	sess, err := s.sessionSvc.Get(agentID, sessionID)
	if err != nil {
		return err
	}

	sess.AddMessages(request)

	//agent
	agt, err := s.agentRepo.Get(agentID)
	if err != nil {
		return err
	}

	// model
	model, err := s.modelRepo.Get(agt.Model())
	if err != nil {
		return err
	}

	// skills
	// openedSkills, err := s.sessSkillRepo.Get(sessionID)
	// if err != nil {
	// 	return err
	// }

	// skills, err := s.skillRegestry.Get(openedSkills...)
	// if err != nil {
	// 	return err
	// }

	toolKit := agt.Tools()
	// for _, s := range skills {
	// 	toolKit = append(toolKit, s.Tools()...)
	// }

	// tools
	tools, err := s.toolRegistry.GetTools(toolKit)
	if err != nil {
		return err
	}

	if providedTools != nil {
		tools = append(tools, providedTools...)
	}

	// sink
	evCh := make(chan runtime.Event, 16)
	go reader.Read(evCh)

	err = s.runtime.RunStream(
		ctx,
		model,
		agt,
		tools,
		sess,
		evCh,
		logging,
	)

	if err != nil {
		return err
	}

	return s.sessionSvc.Save(agt.ID(), sess)
}
