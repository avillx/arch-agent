package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"context"
)

const (
	compactionRatio     = 0.7
	defaultContextLimit = 100_000
	thereshold          = 0.9
)

func doCompact(
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

	var estimateTokens int64 = 0
	for _, m := range tail {
		estimateTokens += estimateMessageTokens(m)
	}
	sess.OverwriteMessages(estimateTokens, tail)

	evCh <- NewCompactionEvent(
		agt.ID(),
		sess.ID(),
		sess.Summary(),
	)

	return nil
}

func shouldCompact(
	inputTokens,
	outputTokens,
	contextLimit int64,
) bool {
	return (inputTokens + outputTokens) >= int64(float64(contextLimit)*thereshold)
}

func estimateMessageTokens(msg agent.Message) int64 {
	// charsPerToken: average characters per token. 4 is a widely-used
	// heuristic for English; slightly overestimates for code/JSON (~3.5).
	const charsPerToken = 4

	// perMessageOverhead: role, ToolCallID, delimiters, etc.
	const perMessageOverhead = 5

	var chars int
	chars += len(msg.Content())

	if agentMessage, ok := msg.(*agent.AgentMessage); ok {
		for _, tc := range agentMessage.ToolCalls() {
			chars += len(tc.ToolName)
			chars += len(string(tc.Arguments))
		}
	}

	if chars == 0 {
		return perMessageOverhead
	}
	return int64(chars/charsPerToken) + perMessageOverhead
}
