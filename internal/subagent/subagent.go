package subagent

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"errors"

	"arch-agent/internal/runtime"

	"context"
)

type ctxKey struct{}

type subAgentCall struct {
	callerAgentID   agent.ID
	recivierAgentID agent.ID
}

const maxSubAgentDepth = 3

func subAgentCallStack(ctx context.Context, s subAgentCall) (context.Context, bool) {
	callStack, _ := ctx.Value(ctxKey{}).([]subAgentCall)
	if len(callStack) >= maxSubAgentDepth {
		return ctx, true
	}
	newStack := append([]subAgentCall{s}, callStack...)
	return context.WithValue(ctx, ctxKey{}, newStack), false
}

type Service struct {
	sessionSvc *session.Service
	chatSvc    *chat.Service
}

func NewService(
	chatSvc *chat.Service,
	sessionSvc *session.Service,
) *Service {
	return &Service{
		chatSvc:    chatSvc,
		sessionSvc: sessionSvc,
	}
}

var ErrCallStackOverflow = errors.New("sub agent call is overflow")

func (s *Service) Call(
	ctx context.Context,
	callerAgentID agent.ID,
	recivierAgentID agent.ID,
	sessionID session.ID,
	request string,
) (string, error) {

	ctx, isOverflow := subAgentCallStack(ctx, subAgentCall{callerAgentID: callerAgentID, recivierAgentID: recivierAgentID})
	if isOverflow {
		return prompt.SubAgentCallStackOverflowCaution(), ErrCallStackOverflow
	}

	subSessID, err := s.sessionSvc.Create(recivierAgentID)
	if err != nil {
		return "", err
	}

	lastAgentMessageContent := ""
	evReader := runtime.EventReader{
		OnComplete: func(i1 agent.ID, i2 session.ID, c *agent.Completion) {
			lastAgentMessageContent = c.Content
		},
	}

	request = prompt.SubAgentGuidance(request)

	err = s.chatSvc.Chat(
		ctx,
		chat.Request{
			AgentID:     recivierAgentID,
			SessionID:   subSessID,
			UserMessage: agent.NewUserMessage(request),
			Reader:      evReader,
		},
	)

	return lastAgentMessageContent, err
}
