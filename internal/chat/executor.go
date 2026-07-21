package chat

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"context"
	"errors"
	"fmt"
)

type Request struct {
	AgentID             agent.ID
	SessionID           session.ID
	UserMessage         *agent.UserMessage
	Reader              runtime.EventReader
	ProvidedToolServers []agent.ToolServer
	Logging             bool
	Additional          string
}

// request executor
type executor struct {
	agentRepo    agent.Repo
	sessionSvc   *session.Service
	modelRepo    agent.ModelRegistry
	toolRegistry agent.ToolRegistry
	harness      *runtime.Harness
	runtime      *runtime.AgentRuntime
}

func NewExecutor(
	agentRepo agent.Repo,
	sessionSvc *session.Service,
	modelRepo agent.ModelRegistry,
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
	toolServers, err := s.toolRegistry.ToolServers(agt.ToolServers()...)
	if err != nil {
		if err := types.DistillErrNotExist(fmt.Sprintf("agent %s", err), err); err != nil {
			return err
		}
	}

	if r.ProvidedToolServers != nil {
		toolServers = append(toolServers, r.ProvidedToolServers...)
	}
	// sink
	evCh := make(chan runtime.Event, 16)
	go r.Reader.Read(evCh)

	err = s.runtime.RunStream(
		ctx,
		runtime.RunStramRequest{
			Model:       model,
			ToolServers: toolServers,
			Sess:        sess,
			Agent:       agt,
			EvCh:        evCh,
			LogActivity: r.Logging,
			Harness:     s.harness,
			BuildContextRequest: runtime.BuildContextRequest{
				IncludeMemory:       true,
				IncludeSkills:       true,
				AllowOptimizeImages: true,
				AddInstuctions:      true,
				Additional:          r.Additional,
			},
		},
	)

	return errors.Join(err, s.sessionSvc.Save(agt.ID(), sess))
}
