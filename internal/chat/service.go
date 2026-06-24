package chat

import (
	"context"
	"fmt"
)

type Service struct {
	executor   *executor
	dispatcher *dispatcher
}

func NewService(e *executor) *Service {
	return &Service{
		executor:   e,
		dispatcher: &dispatcher{processes: map[sessionKey]*requestProcessing{}},
	}
}

func (s *Service) Chat(ctx context.Context, r Request) error {

	if err := validateRequest(r); err != nil {
		return err
	}

	return s.dispatcher.Dispatch(ctx, r, s.executor.chat)
}

func validateRequest(r Request) error {
	if r.AgentID == "" {
		return fmt.Errorf("completion request must include agentID")
	}

	if r.SessionID == "" {
		return fmt.Errorf("completion request must include sessionID")
	}

	if r.UserMessage == nil {
		return fmt.Errorf("completion request must include user message")
	}

	return nil
}
