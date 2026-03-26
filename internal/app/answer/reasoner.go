package answer

import (
	requestcontext "arch-agent/internal/app/executioncontext"
	"arch-agent/internal/domain/conversation"
	"context"
)

type Reasoner interface {
	Reason(
		ctx context.Context,
		params requestcontext.ReasonParams,
	) (ReasonResult, error)
}

type ReasonResult struct {
	toolCalls []conversation.ToolCall
	content   string
}

func NewReasonResult(tc []conversation.ToolCall, content string) ReasonResult {
	return ReasonResult{
		content:   content,
		toolCalls: tc,
	}
}
