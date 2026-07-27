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
	agentRepo  agent.Repo
	runtime    *runtime.AgentRuntime
	toolServer []agent.ToolServer
	model      agent.Model
	harness    *runtime.Harness
}

func NewMemory(
	agentRepo agent.Repo,
	runtime *runtime.AgentRuntime,
	toolServer []agent.ToolServer,
	model agent.Model,
	harness *runtime.Harness,
) (*Memory, error) {

	if !(len(toolServer) > 0) {
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
		model:      model,
		agentRepo:  agentRepo,
		runtime:    runtime,
		toolServer: toolServer,
		harness:    harness,
	}, nil
}

func (m *Memory) ConsolidateImmidate(ctx context.Context, agentID agent.ID, evCh chan runtime.Event) error {
	agt, err := m.agentRepo.Get(agentID)
	if err != nil {
		return nil
	}

	return m.consolidateMemoryFor(ctx, agt, evCh)
}

func (m *Memory) consolidateMemoryFor(ctx context.Context, agt agent.Agent, evCh chan runtime.Event) error {
	return m.runtime.RunStream(
		ctx,
		runtime.RunStramRequest{
			Model:       m.model,
			ToolServers: m.toolServer,
			Sess:        m.createMemorizationSession(agt.ID()),
			Agent:       resolveConsolidationAgent(agt),
			EvCh:        evCh,
			Harness:     m.harness,
			BuildContextRequest: runtime.BuildContextRequest{
				IncludeMemory:       true,
				IncludeSkills:       false,
				AllowOptimizeImages: false,
				AddInstuctions:      true,
			},
		},
	)
}

func (m *Memory) createMemorizationSession(agentID agent.ID) session.Session {
	sess := session.NewSession("hidden")
	sess.AddMessages(agent.NewUserMessage(prompt.MemorizationRequest(agentID)))
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

		evCh := make(chan runtime.Event, 16)
		evReader := runtime.EventReader{}
		go evReader.Read(evCh)

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
			slog.Info("automatic memory consolidation started")

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

func resolveConsolidationAgent(agt agent.Agent) agent.Agent {
	return agent.NewAgent(
		agt.ID(),
		"",
		prompt.Memorization(agt.ID()),
		agt.Model(),
		[]agent.ToolName{},
		nil,
		true,
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
