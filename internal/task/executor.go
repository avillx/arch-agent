package task

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"context"
	"fmt"
	"log/slog"
)

type taskExecutor struct {
	sessionSvc *session.SessionService
	chatSvc    *chat.ChatService
}

// executor
func NewTaskExecutor(
	sessionSvc *session.SessionService,
	chatSvc *chat.ChatService,
) *taskExecutor {
	return &taskExecutor{
		sessionSvc: sessionSvc,
		chatSvc:    chatSvc,
	}
}

// func (s *taskExecutor) Validate(t Task) error {
// 	// get existed agents
// 	agentsCfgs, err := s.agentSvc.List()
// 	if err != nil {
// 		return err
// 	}

// 	// validate agents
// 	agentMap := map[agent.ID]struct{}{}
// 	for _, cfg := range agentsCfgs {
// 		agentMap[cfg.ID()] = struct{}{}
// 	}
// 	for _, agent := range t.Recipients {
// 		if _, ok := agentMap[agent]; !ok {
// 			return fmt.Errorf("agent %s is not exist", agent)
// 		}
// 	}

// 	return nil
// }

func (s *taskExecutor) execute(ctx context.Context, t Task) {
	for _, r := range t.Recipients {
		slog.Info("processing task", "agent", r, "task", t.Name)
		if err := s.processRecipientTask(ctx, agent.ID(r), t.Name, t.Request); err != nil {
			slog.Error("task processing", "agent", r, "task", t.Name, "error", err)
		}
	}
}

func (s *taskExecutor) processRecipientTask(ctx context.Context, agentID agent.ID, taskName, request string) error {

	const autonomusWorking = "Now You working autonomusly if somthing wrong try to contact with someone"

	slog.Info("task executes in background", "task", taskName)

	sessID, err := s.sessionSvc.Create(agentID)
	if err != nil {
		return err
	}

	return s.chatSvc.Chat(ctx, agentID, sessID, fmt.Sprintf("%s\n\n%s", autonomusWorking, request), runtime.EventReader{})
}
