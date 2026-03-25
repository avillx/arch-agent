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
	) (conversation.AgentMessage, error)
}
