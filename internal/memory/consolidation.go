package memory

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/runtime"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type ConsolidatorConfig struct {
	Model       string `toml:"model"`
	Enabled     bool   `toml:"enabled"`
	Instruction string `toml:"instruction"`
}

type MemoryRepo interface {
	Load() (ConsolidatorConfig, error)
	Save(ConsolidatorConfig) error
}

type ConsolidationService struct {
	agentRepo agent.Repo

	modelName string
	model     agent.Model
	modelRepo agent.ModelRegistry

	toolServer []agent.ToolServer
	memoryRepo MemoryRepo

	resolveHooks func(agent.ID) []any
	enabled      bool
	instruction  string

	logger *slog.Logger

	// for lock on set model, enable/disable consolidation and instruction.
	// it is one mutex because no need more. so it any way rare used.
	mu sync.RWMutex
}

func NewConsolidationService(
	agentRepo agent.Repo,
	toolServer []agent.ToolServer,
	hooksReslover func(agent.ID) []any,
	modelRepo agent.ModelRegistry,
	memoryRepo MemoryRepo,
	logger *slog.Logger,
) (*ConsolidationService, error) {

	if !(len(toolServer) > 0) {
		return nil, fmt.Errorf("no tools for managing memory")
	}

	if agentRepo == nil {
		return nil, fmt.Errorf("has no agentRepo")
	}

	svc := &ConsolidationService{
		memoryRepo:   memoryRepo,
		modelRepo:    modelRepo,
		agentRepo:    agentRepo,
		toolServer:   toolServer,
		resolveHooks: hooksReslover,
		enabled:      false,
		logger:       logger.WithGroup("memory"),
	}

	if err := svc.Reload(); err != nil {
		svc.logger.Error("service is not loaded", "error", err)
	}

	return svc, nil
}

func (m *ConsolidationService) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, err := m.memoryRepo.Load()
	if err != nil {
		return err
	}
	model, err := m.modelRepo.Get(cfg.Model)
	if err != nil {
		return err
	}

	m.model = model
	m.instruction = cfg.Instruction
	m.enabled = cfg.Enabled

	return nil
}

func (m *ConsolidationService) ConsolidateImmidate(ctx context.Context, agentID agent.ID, evCh chan runtime.Event) error {
	agt, err := m.agentRepo.Get(agentID)
	if err != nil {
		return err
	}

	return m.consolidateMemoryFor(ctx, agt, evCh)
}

func (m *ConsolidationService) consolidateMemoryFor(ctx context.Context, agt agent.Agent, evCh chan runtime.Event) error {

	model := m.model

	if model == nil {
		return fmt.Errorf("consolidation model is not set")
	}

	m.logger.Info("consolidation started", "agent", agt.ID())

	systemPrompt := prompt.Memorization(agt.ID())
	systemMessage := agent.NewSystemMessage(systemPrompt)

	memoRequest := prompt.MemorizationRequest(agt.ID(), m.Instuction())
	userMessage := agent.NewUserMessage(memoRequest)

	messages := []agent.Message{systemMessage, userMessage}

	tools := []agent.Tool{}
	for _, ts := range m.toolServer {
		tools = append(tools, ts.Tools()...)
	}

	return runtime.RunAgentLoop(
		ctx,
		model,
		messages,
		tools,
		evCh,
		m.resolveHooks(agt.ID()),
	)
}

func (m *ConsolidationService) Enabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.enabled
}

func (m *ConsolidationService) Model() agent.Model {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.model
}

func (m *ConsolidationService) Instuction() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.instruction
}

func (m *ConsolidationService) SetEnabled(state bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = state

	return m.saveConfig()
}

func (m *ConsolidationService) SetModel(modelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	model, err := m.modelRepo.Get(modelName)
	if err != nil {
		return err
	}

	m.model = model
	m.modelName = modelName

	return m.saveConfig()
}

func (m *ConsolidationService) SetInstuction(i string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.instruction = i

	return m.saveConfig()
}

// without mutex
func (m *ConsolidationService) saveConfig() error {
	return m.memoryRepo.Save(ConsolidatorConfig{
		Instruction: m.instruction,
		Model:       m.modelName,
		Enabled:     m.enabled,
	})
}

func (m *ConsolidationService) consolidateInBackground(ctx context.Context) error {
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

func (m *ConsolidationService) Run(ctx context.Context) {
	for {
		ticker := time.Tick(time.Until(nextExecution()))
		select {
		case <-ticker:
			m.logger.Info("automatic consolidation started")

			if !m.Enabled() {
				// unneccecary noise in logs better than silent disable consolidation
				m.logger.Info("skip memory consolidation, consolidation is disabled")
				continue
			}

			if m.Model() == nil {
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
