package answer

import (
	executioncontext "arch-agent/internal/app/executioncontext"
	"arch-agent/internal/domain/conversation"
	"context"
)

type Reasoner interface {
	Reason(
		ctx context.Context,
		params executioncontext.ReasonParams,
	) (*ReasonResult, error)
}

type ReasonResult struct {
	toolCalls []*conversation.ToolCall
	content   string
}

func (res *ReasonResult) Content() string {
	return res.content
}

func (res *ReasonResult) ToolCalls() []*conversation.ToolCall {
	return res.toolCalls
}

func NewReasonResult(tc []*conversation.ToolCall, content string) *ReasonResult {
	return &ReasonResult{
		content:   content,
		toolCalls: tc,
	}
}
