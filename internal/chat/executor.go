package chat

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"context"
	"errors"
)

type Request struct {
	AgentID       agent.ID
	SessionID     session.ID
	UserMessage   *agent.UserMessage
	Reader        runtime.EventReader
	ProvidedTools []agent.Tool
	Logging       bool
}

// request executor
type executor struct {
	agentRepo    agent.Repo
	sessionSvc   *session.Service
	modelRepo    agent.ModelRepository
	toolRegistry agent.ToolRegistry
	runtime      *runtime.AgentRuntime
}

func NewExecutor(
	agentRepo agent.Repo,
	sessionSvc *session.Service,
	modelRepo agent.ModelRepository,
	toolRegistry agent.ToolRegistry,
	runtime *runtime.AgentRuntime,
) *executor {
	return &executor{
		agentRepo:    agentRepo,
		sessionSvc:   sessionSvc,
		modelRepo:    modelRepo,
		toolRegistry: toolRegistry,
		runtime:      runtime,
	}
}

func (s *executor) chat(
	ctx context.Context,
	r Request,
) error {

	// session
	sess, err := s.sessionSvc.Get(r.AgentID, r.SessionID)
	if err != nil {
		return err
	}

	sess.AddMessages(r.UserMessage)

	//agent
	agt, err := s.agentRepo.Get(r.AgentID)
	if err != nil {
		return err
	}

	// model
	model, err := s.modelRepo.Get(agt.Model())
	if err != nil {
		return err
	}

	// tools
	tools, err := s.toolRegistry.GetTools(agt.Tools())
	if err != nil {
		return err
	}

	if r.ProvidedTools != nil {
		tools = append(tools, r.ProvidedTools...)
	}

	// sink
	evCh := make(chan runtime.Event, 16)
	go r.Reader.Read(evCh)

	err = s.runtime.RunStream(
		ctx,
		model,
		agt,
		tools,
		sess,
		evCh,
		r.Logging,
	)

	return errors.Join(err, s.sessionSvc.Save(agt.ID(), sess))
}
