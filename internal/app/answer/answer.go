package answer

import (
	"arch-agent/internal/app/executioncontext"
	"arch-agent/internal/app/memory"
	"arch-agent/internal/app/session"
	tools "arch-agent/internal/app/toolexecutor"
	"context"
	"errors"
	"time"
)

const ExecutionTimeLimin = 40 * time.Second

type AnswerUCLogger interface {
	Error(string, ...any)
	Debug(string, ...any)
	Info(string, ...any)
}

type AnswerUseCase struct {
	reasoner                Reasoner
	sessionService          *session.SessionService
	agentRepo               executioncontext.AgentRepository
	executionContextFactory *executioncontext.RequestContextFactory
	memoryService           *memory.MemoryService
	logger                  AnswerUCLogger
}

func NewAnswerUseCase(
	r Reasoner,
	ss *session.SessionService,
	ms *memory.MemoryService,
	ar executioncontext.AgentRepository,
	ecf *executioncontext.RequestContextFactory,
	l AnswerUCLogger,
) *AnswerUseCase {
	return &AnswerUseCase{
		reasoner:                r,
		sessionService:          ss,
		agentRepo:               ar,
		executionContextFactory: ecf,
		memoryService:           ms,
		logger:                  l,
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

	runningMemory, err := a.memoryService.RunningMemory()
	errs = errors.Join(errs, err)

	session, err := a.sessionService.Session()
	if err != nil {
		return err
	}
	session.AddUserMessage(request)
	executionContext, err := a.executionContextFactory.Build(
		ctx,
		a.agentRepo.Get(),
		runningMemory,
		session.Messages(),
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

		reasonParams := executionContext.NextReasonParams(ctx, session.Messages())
		a.logger.Debug("reason params created", "reflection", reasonParams.Reflection)

		reasonResultValue, err := a.reasoner.Reason(ctx, reasonParams)
		if err != nil {
			return err
		}

		reasonContent := reasonResultValue.content
		toolCalls := reasonResultValue.toolCalls
		a.logger.Debug("reason is complete", "result", reasonResultValue)

		session.AddAgentMessage(reasonContent, toolCalls)

		// resolve content
		if reasonContent != "" {
			if err := contentRecivier(ctx, reasonContent); err != nil {
				return errors.Join(errs, err)
			}
		}

		// tool calls
		toolExecutionResult, err := te.Execute(ctx, toolCalls)
		errs = errors.Join(errs, err)
		session.AddToolCallResults(toolExecutionResult)

		if !executionContext.ShouldFollowUp(toolCalls) {
			break
		}

	}

	err = a.sessionService.Close(ctx, session)

	return errors.Join(errs, err)
}
