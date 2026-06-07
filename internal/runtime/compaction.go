package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"context"
)

const compactionRatio = 0.7
const defaultContextLimit = 100_000

type Compactor struct {
	tokenCounter agent.TokenCounter
}

func NewCompactor(tc agent.TokenCounter) *Compactor {
	return &Compactor{
		tokenCounter: tc,
	}
}

func (c *Compactor) doCompact(
	ctx context.Context,
	sess session.Session,
	agt agent.Agent,
	model agent.Model,
	evCh chan Event,
) error {

	msgs := sess.Messages()

	divide := int(float32(len(msgs)) * compactionRatio)
	toCompact := msgs[:divide]
	tail := msgs[divide:]

	completion, err := model.Complete(ctx, nil,
		[]agent.Message{
			agent.NewSystemMessage(prompt.CompactionPrompt()),
			agent.NewUserMessage(agent.StringifyConversation(toCompact)),
		},
	)
	if err != nil {
		return err
	}

	sess.AddSummary(completion.Content)
	sess.OverwriteMessages(tail)

	evCh <- NewCompactionEvent(
		agt.ID(),
		sess.ID(),
		sess.Summary(),
	)

	return nil
}

func (c *Compactor) shouldCompact(
	sess session.Session,
	additionalMessages []agent.Message,
	tools []agent.Tool,
	model agent.Model,
) bool {

	modelContextLimit := model.ContextLimit()
	if modelContextLimit == 0 {
		modelContextLimit = defaultContextLimit
	}
	thereshold := modelContextLimit * 90 / 100

	tokenCount := sess.MessagesTokens()
	if len(additionalMessages) > 0 {
		tokenCount += c.tokenCounter.Messages(additionalMessages)
	}
	if len(tools) > 0 {
		tokenCount += c.tokenCounter.Tools(tools)
	}

	return tokenCount >= thereshold
}
