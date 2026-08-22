package memory

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/runtime"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type InstructedFSServer struct {
	agent.ToolServer
	cwd string
}

func NewInstuctFS(cwd string, srv agent.ToolServer) *InstructedFSServer {
	return &InstructedFSServer{
		ToolServer: srv,
		cwd:        cwd,
	}
}

func (r *InstructedFSServer) AgentInstruction(agt agent.Agent) string {
	return prompt.ConsolidationFSInstruction(r.cwd, agt.ID())
}

type Memory struct {
	agentRepo    agent.Repo
	toolServer   []agent.ToolServer
	model        agent.Model
	resolveHooks func(agent.ID) []any
	logger       *slog.Logger
}

func NewMemory(
	agentRepo agent.Repo,
	toolServer []agent.ToolServer,
	model agent.Model,
	hooksReslover func(agent.ID) []any,
	logger *slog.Logger,
) (*Memory, error) {

	if !(len(toolServer) > 0) {
		return nil, fmt.Errorf("no tools for managing memory")
	}

	if model == nil {
		return nil, fmt.Errorf("has no model")
	}

	if agentRepo == nil {
		return nil, fmt.Errorf("has no agentRepo")
	}

	return &Memory{
		model:        model,
		agentRepo:    agentRepo,
		toolServer:   toolServer,
		resolveHooks: hooksReslover,
		logger:       logger.WithGroup("memory"),
	}, nil
}

func (m *Memory) ConsolidateImmidate(ctx context.Context, agentID agent.ID, evCh chan runtime.Event) error {
	agt, err := m.agentRepo.Get(agentID)
	if err != nil {
		return err
	}

	return m.consolidateMemoryFor(ctx, agt, evCh)
}

func (m *Memory) consolidateMemoryFor(ctx context.Context, agt agent.Agent, evCh chan runtime.Event) error {

	m.logger.Info("consolidation started", "agent", agt.ID())

	systemPrompt := prompt.Memorization(agt.ID())
	systemMessage := agent.NewSystemMessage(systemPrompt)

	memoRequest := prompt.MemorizationRequest(agt.ID())
	userMessage := agent.NewUserMessage(memoRequest)

	messages := []agent.Message{systemMessage, userMessage}

	tools := []agent.Tool{}
	for _, ts := range m.toolServer {
		tools = append(tools, ts.Tools()...)
	}

	return runtime.RunAgentLoop(
		ctx,
		m.model,
		messages,
		tools,
		evCh,
		m.resolveHooks(agt.ID()),
	)
}

func (m *Memory) SetModel(model agent.Model) {
	m.model = model
}

func (m *Memory) consolidateInBackground(ctx context.Context) error {
	agents, err := m.agentRepo.All()
	if err != nil {
		return err
	}

	var errc error
	for _, agt := range agents {
		if !agt.HasMemory() {
			continue
		}

		evCh := make(chan runtime.Event, 16)
		defer close(evCh)

		// for drop consolidation events
		go func() {
			for range evCh {
			}
		}()

		if err := m.consolidateMemoryFor(ctx, agt, evCh); err != nil {
			errc = errors.Join(errc, err)
		}
	}

	return errc
}

func (m *Memory) Run(ctx context.Context) {
	for {
		ticker := time.Tick(time.Until(nextExecution()))
		select {
		case <-ticker:
			m.logger.Info("automatic consolidation started")

			if m.model == nil {
				m.logger.Warn("consolidation declined, model is not defined")
				continue
			}

			if err := m.consolidateInBackground(ctx); err != nil {
				m.logger.Error("consolidation", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func nextExecution() time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
