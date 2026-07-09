package memory

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type Memory struct {
	agentRepo agent.Repo
	runtime   *runtime.AgentRuntime
	tools     []agent.Tool
	model     agent.Model
}

func NewMemory(
	agentRepo agent.Repo,
	runtime *runtime.AgentRuntime,
	tools []agent.Tool,
	model agent.Model,
) (*Memory, error) {

	if !(len(tools) > 0) {
		return nil, fmt.Errorf("no tools for managing memory")
	}

	if model == nil {
		return nil, fmt.Errorf("has no model")
	}

	if runtime == nil {
		return nil, fmt.Errorf("has no agent runtime")
	}

	if agentRepo == nil {
		return nil, fmt.Errorf("has no agentRepo")
	}

	return &Memory{
		model:     model,
		agentRepo: agentRepo,
		runtime:   runtime,
		tools:     tools,
	}, nil
}

func (m *Memory) DreamImmidate(ctx context.Context, agentID agent.ID) error {
	agt, err := m.agentRepo.Get(agentID)
	if err != nil {
		return nil
	}

	return m.consolidateMemoryFor(ctx, agt)
}

func (m *Memory) consolidateMemoryFor(ctx context.Context, agt agent.Agent) error {
	evCh := make(chan runtime.Event, 16)
	evReader := runtime.EventReader{}
	go evReader.Read(evCh)

	return m.runtime.RunStream(
		ctx,
		m.model,
		m.buildConsolidationAgent(agt),
		m.tools,
		m.createMemorizationSession(agt.ID()),
		evCh,
		false,
		nil,
	)
}

func (m *Memory) createMemorizationSession(agentID agent.ID) session.Session {
	sess := session.NewSession("hidden")
	sess.AddMessages(agent.NewUserMessage(prompt.GetMemorizationRequest(agentID)))
	return sess
}

func (m *Memory) SetModel(model agent.Model) {
	m.model = model
}

func (m *Memory) consolidateMemory(ctx context.Context) error {
	agents, err := m.agentRepo.All()
	if err != nil {
		return err
	}

	var errc error
	for _, agt := range agents {
		if !agt.HasMemory() {
			continue
		}

		if err := m.consolidateMemoryFor(ctx, agt); err != nil {
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
			if m.model == nil {
				slog.Warn("memorization declined", "warninig", "memory model is not setted")
				continue
			}

			if err := m.consolidateMemory(ctx); err != nil {
				slog.Error("memory consolidation", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *Memory) buildConsolidationAgent(agt agent.Agent) agent.Agent {
	return agent.NewAgent(
		agt.ID(),
		"",
		prompt.GetMemorizationPrompt(agt.ID()),
		agt.Model(),
		[]agent.ToolName{},
		nil,
		nil,
		false,
	)
}

func nextExecution() time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
