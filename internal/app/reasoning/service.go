package reasoning

import (
	"arch-agent/internal/app/types"
	"context"
	"errors"
	"log/slog"
	"slices"
)

type ToolCallRecivier interface {
	ReciveCall(ctx context.Context, call *types.ToolCall) (string, error)
	Tools() ([]types.ToolDefinition, error)
}

type reasoner interface {
	Reason(
		ctx context.Context,
		prompt string,
		toolDefs []types.ToolDefinition,
		messages []types.Message,
	) (*ReasonResult, error)
}

type ReasonResult struct {
	ToolCalls []*types.ToolCall
	Content   string
	Done      bool
}

type Service struct {
	recallBudget    int
	reasoner        reasoner
	toolCallReciver ToolCallRecivier
}

func NewService(
	recallBudget int,
	reasoner reasoner,
	toolCallReciver ToolCallRecivier,
) *Service {
	return &Service{
		recallBudget:    recallBudget,
		reasoner:        reasoner,
		toolCallReciver: toolCallReciver,
	}
}

func (s *Service) Reason(
	ctx context.Context,
	prompt string,
	contentRecivier func(ctx context.Context, content string) error,
	msgs []types.Message,
) ([]types.Message, error) {

	newMsgs := []types.Message{}

	for toolCallBudget := 0; toolCallBudget <= s.recallBudget; toolCallBudget++ {

		done, processResultMsgs, err := s.processReason(ctx, prompt, contentRecivier, slices.Concat(msgs, newMsgs))
		newMsgs = slices.Concat(newMsgs, processResultMsgs)

		if err != nil {
			return newMsgs, err
		}

		if done {
			return newMsgs, nil
		}
	}

	return newMsgs, errors.New("recall budget expires")
}

func (s *Service) processReason(
	ctx context.Context,
	prompt string,
	contentRecivier func(ctx context.Context, content string) error,
	msgs []types.Message,
) (bool, []types.Message, error) {

	toolDefs, err := s.toolCallReciver.Tools()
	if err != nil {
		return true, nil, err
	}

	result, err := s.reasoner.Reason(ctx, prompt, toolDefs, msgs)
	if err != nil {
		return true, nil, err
	}
	slog.Debug("reasoning processed", "value", result)

	newMsgs := []types.Message{
		types.NewAgentMessage(result.Content, result.ToolCalls),
	}

	toolResultMsgs, err := s.processResult(ctx, result, contentRecivier)
	if err != nil {
		return true, nil, err
	}

	if len(toolResultMsgs) > 0 {
		newMsgs = slices.Concat(newMsgs, toolResultMsgs)
	}

	return result.Done, newMsgs, err
}

func (s *Service) processResult(
	ctx context.Context,
	result *ReasonResult,
	contentRecivier func(ctx context.Context, content string) error,
) ([]types.Message, error) {
	// d.useCaselogger.Debug("dreaming itterated", &result)
	var errc error

	if contentRecivier != nil && result.Content != "" {
		if err := contentRecivier(ctx, result.Content); err != nil {
			errc = errors.Join(errc, err)
		}
	}
	newMsg, err := s.processToolCalls(ctx, result.ToolCalls)
	errc = errors.Join(errc, err)

	return newMsg, errc
}

func (s *Service) processToolCalls(ctx context.Context, calls []*types.ToolCall) ([]types.Message, error) {
	messages := []types.Message{}

	// TODO:
	// more elegant nil reciviers processing
	for _, call := range calls {
		msg, err := s.processToolCall(ctx, call)
		if msg != nil {
			messages = append(messages, msg)
		}
		if err != nil {
			return messages, err
		}
	}

	return messages, nil
}

func (s *Service) processToolCall(ctx context.Context, call *types.ToolCall) (types.Message, error) {
	if s.toolCallReciver == nil {
		slog.Error("model call tool, but have not tool call Recivier", "tool", call.ToolName(), "args", call.Arguments())
		return nil, nil
	}

	result, err := s.toolCallReciver.ReciveCall(ctx, call)
	if err != nil && !errors.Is(err, ErrAgentToolCallMistake) {
		return nil, err
	}
	if err != nil {
		result += err.Error()
	}
	return types.NewToolResultMessage(call.ID(), result), nil
}

var ErrAgentToolCallMistake = errors.New("agent bad toolcall")
