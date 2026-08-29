package task

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"context"
	"log/slog"
)

type executor struct {
	sessionSvc   *session.Service
	chatExecutor chat.ChatExecutor
	logger       *slog.Logger
}

func NewExecutor(
	sessionSvc *session.Service,
	chatExecutor chat.ChatExecutor,
	logger *slog.Logger,
) *executor {
	return &executor{
		sessionSvc:   sessionSvc,
		chatExecutor: chatExecutor,
		logger:       logger,
	}
}

func (s *executor) execute(ctx context.Context, t TaskConfig) {

	logger := s.logger.With("task", t.Name)

	for _, r := range t.Recipients {
		logger := logger.With("agent", r)

		logger.Info("processing started")
		if err := s.processRecipientTask(ctx, agent.ID(r), t.Request); err != nil {
			logger.Error("processing", "error", err)
		}
	}
}

func (s *executor) processRecipientTask(ctx context.Context, agentID agent.ID, request string) error {

	sessID, err := s.sessionSvc.Create(agentID, prompt.GetAutonomusGuidance())
	if err != nil {
		return err
	}

	return s.chatExecutor.Chat(
		ctx,
		chat.Request{
			AgentID:     agentID,
			SessionID:   sessID,
			UserMessage: agent.NewUserMessage(request),
			Logging:     true,
		},
	)
}
