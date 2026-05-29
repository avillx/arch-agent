package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"context"
)

const compactionRatio = 0.7

type CompactionRequest struct {
	sess  session.Session
	agt   agent.Agent
	model agent.Model
}

func doCompact(ctx context.Context, req CompactionRequest) error {

	msgs := req.sess.Messages()

	divide := int(float32(len(msgs)) * compactionRatio)
	toCompact := msgs[:divide]
	tail := msgs[divide:]

	completion, err := req.model.Complete(ctx, nil,
		[]agent.Message{
			agent.NewSystemMessage(prompt.CompactionPrompt()),
			agent.NewUserMessage(agent.StringifyConversation(toCompact)),
		},
	)
	if err != nil {
		return err
	}

	req.sess.AddSummary(completion.Content)
	req.sess.OverwriteMessages(tail)

	return nil
}

func shouldCompact(sess session.Session, model agent.Model) bool {
	return sess.Tokens() >= (model.ContextLimit() * 90 / 100)
}
