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
	sessionSvc *session.Service
	chatSvc    *chat.Service
}

// executor
func NewExecutor(
	sessionSvc *session.Service,
	chatSvc *chat.Service,
) *executor {
	return &executor{
		sessionSvc: sessionSvc,
		chatSvc:    chatSvc,
	}
}

func (s *executor) execute(ctx context.Context, t TaskConfig) {
	for _, r := range t.Recipients {
		slog.Info("processing task", "agent", r, "task", t.Name)
		if err := s.processRecipientTask(ctx, agent.ID(r), t.Name, t.Request); err != nil {
			slog.Error("task processing", "agent", r, "task", t.Name, "error", err)
		}
	}
}

func (s *executor) processRecipientTask(ctx context.Context, agentID agent.ID, taskName, request string) error {
	slog.Info("task executes in background", "task", taskName)

	sessID, err := s.sessionSvc.Create(agentID)
	if err != nil {
		return err
	}

	return s.chatSvc.Chat(
		ctx,
		chat.Request{
			AgentID:     agentID,
			SessionID:   sessID,
			UserMessage: agent.NewUserMessage(prompt.GetAutonomusRequest(request)),
			Logging:     true,
		},
	)
}
