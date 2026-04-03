package answer

import (
	"arch-agent/internal/app/executioncontext"
	"arch-agent/internal/app/message"
	"context"
)

type Reasoner interface {
	Reason(
		ctx context.Context,
		params executioncontext.ReasonParams,
	) (*ReasonResult, error)
}

type ReasonResult struct {
	toolCalls []*message.ToolCall
	content   string
}

func (res *ReasonResult) Content() string {
	return res.content
}

func (res *ReasonResult) ToolCalls() []*message.ToolCall {
	return res.toolCalls
}

func NewReasonResult(tc []*message.ToolCall, content string) *ReasonResult {
	return &ReasonResult{
		content:   content,
		toolCalls: tc,
	}
}
