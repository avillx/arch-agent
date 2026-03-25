package answer

import (
	requestcontext "arch-agent/internal/app/executioncontext"
	tools "arch-agent/internal/app/toolexecutor"
	"arch-agent/internal/domain/conversation"
	"context"
	"errors"
	"time"
)

const ExecutionTimeLimin = 3 * time.Minute

type ContentRecivier interface {
	Send(ctx context.Context, content string) error
}

type ConversationRespository interface {
	Get() *conversation.Conversation
	Save([]conversation.Message)
	Optimize()
}

type AnswerUseCase struct {
	reasoner                Reasoner
	conversationRepo        ConversationRespository
	agentRepo               requestcontext.AgentRepository
	executionContextFactory *requestcontext.RequestContextFactory
}

func (a *AnswerUseCase) Execute(
	ctx context.Context,
	cr ContentRecivier,
	te *tools.Executor,
	cmd *AnswerCommand,
) error {
	var errs error

	ctx, cancel := context.WithTimeout(ctx, ExecutionTimeLimin)
	defer cancel()

	conver := a.conversationRepo.Get()
	conver.AddUserMessage(cmd.Content)
	executionContext, err := a.executionContextFactory.Build(
		ctx,
		a.agentRepo.Get(),
		conver.Messages(),
		cmd.toolDefs)
	if err != nil {
		return err
	}

	reason := true
	for reason {

		select {
		case <-ctx.Done():
			errs = errors.Join(errs, errors.New("execution time exceed"))
			reason = false
			continue
		default:
		}

		reasonParams := executionContext.NextReasonParams(ctx, conver.Messages())

		reasonResult, err := a.reasoner.Reason(ctx, reasonParams)
		if err != nil {
			return err
		}

		reasonContent := reasonResult.Content()
		toolCalls := reasonResult.ToolCalls()

		conver.AddAgentMessage(reasonContent, toolCalls)

		// resolve content
		if err := cr.Send(ctx, reasonContent); err != nil {
			return errors.Join(errs, err)
		}

		// tool calls
		toolExecutionResult, err := te.Execute(ctx, toolCalls)
		errs = errors.Join(errs, err)
		conver.AddToolCallResults(toolExecutionResult)

		if !executionContext.ShouldFollowUp(toolCalls) {
			break
		}

	}

	a.conversationRepo.Save(conver.NewMessages())

	if conver.IsOverflow() {
		a.conversationRepo.Optimize()
	}

	return errs
}
