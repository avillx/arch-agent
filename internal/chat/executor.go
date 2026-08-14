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

type EventCallbacks struct {
	OnError      func(agent.ID, session.ID, error)
	OnComplete   func(agent.ID, session.ID, *agent.Completion)
	OnToolResult func(agent.ID, session.ID, *agent.ToolResult)
	OnCompaction func(agent.ID, session.ID, string)
	OnEvent      func(runtime.Event)
}

type Request struct {
	AgentID             agent.ID
	SessionID           session.ID
	UserMessage         *agent.UserMessage
	ProvidedToolServers []agent.ToolServer
	Logging             bool
	EventCallbacks
}

// request executor
type executor struct {
	agentRepo    agent.Repo
	sessionSvc   *session.Service
	modelRepo    agent.ModelRegistry
	toolRegistry agent.ToolRegistry
	observer     *Observer
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
	observer *Observer,
) *executor {
	return &executor{
		agentRepo:    agentRepo,
		sessionSvc:   sessionSvc,
		modelRepo:    modelRepo,
		toolRegistry: toolRegistry,
		runtime:      runtime,
		harness:      harness,
		observer:     observer,
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

	// logging
	if r.Logging {

		// commit user message
		s.observer.Commit(agt.ID(), sess.ID(), []agent.Message{r.UserMessage})

		// wrap on complete for log shit
		prev := r.OnComplete
		r.OnComplete = func(agentID agent.ID, sessID session.ID, c *agent.Completion) {
			msg := agent.NewAgentMessage(c.Content, nil)
			msgs := []agent.Message{msg}
			s.observer.Commit(agentID, sessID, msgs)
			prev(agentID, sessID, c)
		}
	}

	// sink
	evCh := make(chan runtime.Event, 16)
	defer close(evCh)
	go ReadEvents(
		evCh,
		r.OnError,
		r.OnComplete,
		r.OnToolResult,
		r.OnCompaction,
		r.OnEvent,
	)

	// run
	streamErr := s.runtime.RunStream(
		ctx,
		runtime.RunStramRequest{
			Model:       model,
			ToolServers: toolServers,
			Sess:        sess,
			Agent:       agt,
			EvCh:        evCh,
			Harness:     s.harness,
			BuildContextRequest: runtime.BuildContextRequest{
				IncludeMemory:       true,
				IncludeSkills:       true,
				AllowOptimizeImages: true,
				AddInstuctions:      true,
			},
		},
	)

	// save session
	sessErr := s.sessionSvc.Save(agt.ID(), sess)

	return errors.Join(streamErr, sessErr)
}

func ReadEvents(
	ch <-chan runtime.Event,
	OnError func(agent.ID, session.ID, error),
	OnComplete func(agent.ID, session.ID, *agent.Completion),
	OnToolResult func(agent.ID, session.ID, *agent.ToolResult),
	OnCompaction func(agent.ID, session.ID, string),
	OnEvent func(runtime.Event),
) {
	for ev := range ch {
		if OnEvent != nil {
			OnEvent(ev)
		}

		switch typedEv := ev.(type) {
		case runtime.ErrEvent:
			if OnError != nil {
				OnError(typedEv.Agent(), typedEv.Session(), typedEv.Err())
			}
		case runtime.CompleteEvent:
			if OnComplete != nil {
				OnComplete(typedEv.Agent(), typedEv.Session(), typedEv.Complete())
			}
		case runtime.ToolCallResultEvent:
			if OnToolResult != nil {
				OnToolResult(typedEv.Agent(), typedEv.Session(), typedEv.Result())
			}
		case runtime.CompactionEvent:
			if OnCompaction != nil {
				OnCompaction(ev.Agent(), ev.Session(), typedEv.Summary())
			}
		}
	}
}
