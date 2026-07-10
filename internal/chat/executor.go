package chat

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"context"
	"errors"
	"log/slog"
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
	harness      *runtime.Harness
	runtime      *runtime.AgentRuntime
}

func NewExecutor(
	agentRepo agent.Repo,
	sessionSvc *session.Service,
	modelRepo agent.ModelRepository,
	toolRegistry agent.ToolRegistry,
	runtime *runtime.AgentRuntime,
	harness *runtime.Harness,
) *executor {
	return &executor{
		agentRepo:    agentRepo,
		sessionSvc:   sessionSvc,
		modelRepo:    modelRepo,
		toolRegistry: toolRegistry,
		runtime:      runtime,
		harness:      harness,
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
	tools, err := s.toolRegistry.GetServerTools(agt.ToolServers())
	if err != nil {
		if err := distillErrNotExist(agt.ID(), err); err != nil {
			return err
		}
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
		s.harness,
	)

	return errors.Join(err, s.sessionSvc.Save(agt.ID(), sess))
}

func distillErrNotExist(agentId agent.ID, err error) error {
	errWrapper, ok := err.(interface{ Unwrap() []error })
	if !ok {
		if errors.Is(err, types.ErrIsNotExist) {
			slog.Warn("chat process", "agent", agentId, "error", err)
			return nil
		}
		return err
	}

	var unExpectedErrs []error
	for _, werr := range errWrapper.Unwrap() {
		if errors.Is(werr, types.ErrIsNotExist) {
			slog.Warn("chat process", "agent", agentId, "error", werr)
			continue
		}

		unExpectedErrs = append(unExpectedErrs, err)
	}
	return errors.Join(unExpectedErrs...)
}
