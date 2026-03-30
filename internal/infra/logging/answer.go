package logging

import (
	"arch-agent/internal/app/answer"
	"arch-agent/internal/domain/conversation"
	"log/slog"
)

type AnswerUCLogger struct{}

func (*AnswerUCLogger) Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

func (*AnswerUCLogger) Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func (*AnswerUCLogger) Debug(msg string, args ...any) {
	for i, arg := range args {
		switch typedArg := arg.(type) {
		case *answer.ReasonResult:
			args[i] = NewLogWrappedReasonResult(typedArg)
		}
	}

	slog.Debug(msg, args...)
}

type LogWrappedReasonResult struct {
	Content   string
	ToolCalls []slog.Attr
}

func NewLogWrappedReasonResult(res *answer.ReasonResult) *LogWrappedReasonResult {
	rawToolCalls := res.ToolCalls()
	wrappedToolCalls := make([]slog.Attr, 0, len(rawToolCalls))

	for _, rawTC := range rawToolCalls {
		wrappedToolCalls = append(wrappedToolCalls, slog.Any(rawTC.ToolName(), &LogWrappedToolCall{rawTC}))
	}

	return &LogWrappedReasonResult{
		Content:   res.Content(),
		ToolCalls: wrappedToolCalls,
	}
}

func (tc *LogWrappedReasonResult) LogValue() slog.Value {
	attrs := append([]slog.Attr{slog.String("content", tc.Content)}, tc.ToolCalls...)
	return slog.GroupValue(attrs...)
}

type LogWrappedToolCall struct {
	*conversation.ToolCall
}

func (tc *LogWrappedToolCall) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("ID", tc.ID()),
		slog.String("args", string(tc.ToolCall.Arguments())),
	)
}
