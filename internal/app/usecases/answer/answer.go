package answer

import (
	"arch-agent/internal/app/reasoning"
	"arch-agent/internal/app/session"
	"context"
	"time"
)

const ExecutionTimeLimit = 2 * time.Minute

type AnswerUseCase struct {
	reasoningService *reasoning.Service
	sessionService   *session.Service
	contextAssembler *ContextAssembler
}

func NewAnswerUseCase(
	r *reasoning.Service,
	ss *session.Service,
	ca *ContextAssembler,
) *AnswerUseCase {
	return &AnswerUseCase{
		reasoningService: r,
		sessionService:   ss,
		contextAssembler: ca,
	}
}

func (a *AnswerUseCase) Execute(
	ctx context.Context,
	request string,
	contentRecivier func(ctx context.Context, content string) error,
	contextDescription string,
) error {

	// ctx, cancel := context.WithTimeout(ctx, ExecutionTimeLimit)
	// defer cancel()

	session, err := a.sessionService.Session()
	if err != nil {
		return err
	}

	session.AddUserMessage(request)

	prompt, err := a.contextAssembler.BuildPrompt(ctx, contextDescription, session.Messages())
	if err != nil {
		return err
	}

	newMsgs, err := a.reasoningService.Reason(ctx, prompt, contentRecivier, session.Messages())
	if err != nil {
		return err
	}

	session.AddMessages(newMsgs)

	return a.sessionService.Close(ctx, session)
}
