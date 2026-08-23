package subagent

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/prompt"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"errors"
	"fmt"
	"log/slog"

	"context"
)

var ErrCallStackOverflow = errors.New("sub agent call is overflow")

const maxSubAgentDepth = 3

type Service struct {
	chatExecutor chat.ChatExecutor
	sessService  *session.Service
	logger       *slog.Logger
}

func NewService(
	chatExecutor chat.ChatExecutor,
	sessService *session.Service,
	logger *slog.Logger,
) *Service {
	return &Service{
		chatExecutor: chatExecutor,
		sessService:  sessService,
		logger:       logger.WithGroup("sub_agent"),
	}
}

func (s *Service) Call(
	ctx context.Context,
	subAgentID agent.ID,
	request string,
) (string, error) {

	ctx, isOverflow := subAgentCallStack(ctx, subAgentCall{subagent: subAgentID})
	if isOverflow {
		return "", ErrCallStackOverflow
	}

	// create session
	sessID, err := s.sessService.Create(subAgentID, prompt.SubAgentGuidance())
	if err != nil {
		return "", err
	}

	logger := s.logger.With("sub_agent", subAgentID, "session", sessID)

	// sink
	lastAgentMessageContent := ""

	// event callbacks

	onLoopExit := func(lee *runtime.LoopExitEvent) {
		if lee.Err() != nil {
			message := fmt.Sprintf("sub agent %s, fall with error", subAgentID)
			lastAgentMessageContent += message

			logger.Error(
				"fall with error",
				"error", err,
			)
		}
	}

	onComplete := func(ce *runtime.CompleteEvent) {
		lastAgentMessageContent = ce.Complete().Content
	}

	eventCallbacks := chat.EventCallbacks{
		OnComplete:        onComplete,
		OnLoopExit:        onLoopExit,
		OnCompleteMistake: func(cme *runtime.CompletionMistakeEvent) {},
		OnToolResult:      func(tre *runtime.ToolResultEvent) {},
		OnCompaction:      func(ce *runtime.CompactionEvent) {},
		OnToolErr:         func(tcee *runtime.ToolCallErrEvent) {},
		OnEvent:           func(e runtime.Event) {},
	}

	logger.Info("running")

	// if sub agent loop exit with error
	// agent alrady recieve message about issues
	// cause alredy logged
	s.chatExecutor.Chat(
		ctx,
		chat.Request{
			AgentID:        subAgentID,
			SessionID:      sessID,
			UserMessage:    agent.NewUserMessage(request),
			EventCallbacks: eventCallbacks,
			Logging:        false,
		},
	)

	return lastAgentMessageContent, nil
}

type ctxKey struct{}

type subAgentCall struct {
	subagent agent.ID
}

func subAgentCallStack(ctx context.Context, s subAgentCall) (context.Context, bool) {
	callStack, _ := ctx.Value(ctxKey{}).([]subAgentCall)
	if len(callStack) >= maxSubAgentDepth {
		return ctx, true
	}
	newStack := append([]subAgentCall{s}, callStack...)
	return context.WithValue(ctx, ctxKey{}, newStack), false
}
