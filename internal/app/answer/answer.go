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

type ConversationRespository interface {
	Get() *conversation.Conversation
	Save([]conversation.Message)
	Optimize()
}
type AnswerUCLogger interface {
	Error(string, ...any)
	Debug(string, ...any)
	Info(string, ...any)
}

type AnswerUseCase struct {
	reasoner                Reasoner
	conversationRepo        ConversationRespository
	agentRepo               requestcontext.AgentRepository
	executionContextFactory *requestcontext.RequestContextFactory
	logger                  AnswerUCLogger
}

func NewAnswerUseCase(
	reasoner Reasoner,
	conversationRepo ConversationRespository,
	agentRepo requestcontext.AgentRepository,
	executionContextFactory *requestcontext.RequestContextFactory,
	logger AnswerUCLogger,
) *AnswerUseCase {
	return &AnswerUseCase{
		reasoner:                reasoner,
		conversationRepo:        conversationRepo,
		agentRepo:               agentRepo,
		executionContextFactory: executionContextFactory,
		logger:                  logger,
	}
}

func (a *AnswerUseCase) Execute(
	ctx context.Context,
	request string,
	contentRecivier func(ctx context.Context, content string) error,
	contextDescription string,
	tcr tools.ToolCallRecivier,
) error {
	var errs error

	te := tools.NewExecutor(tcr)

	ctx, cancel := context.WithTimeout(ctx, ExecutionTimeLimin)
	defer cancel()

	conver := a.conversationRepo.Get()
	conver.AddUserMessage(request)
	executionContext, err := a.executionContextFactory.Build(
		ctx,
		a.agentRepo.Get(),
		conver.Messages(),
		contextDescription,
		tcr.Tools())
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
		a.logger.Debug("reason params created", "reflection", reasonParams.Reflection)

		reasonResultValue, err := a.reasoner.Reason(ctx, reasonParams)
		if err != nil {
			return err
		}

		reasonContent := reasonResultValue.content
		toolCalls := reasonResultValue.toolCalls
		a.logger.Debug("reason is complete", "result", reasonResultValue)

		conver.AddAgentMessage(reasonContent, toolCalls)

		// resolve content
		if reasonContent != "" {
			if err := contentRecivier(ctx, reasonContent); err != nil {
				return errors.Join(errs, err)
			}
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
