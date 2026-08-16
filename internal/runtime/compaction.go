package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"context"
	"fmt"
)

const (
	compactionRatio     = 0.7
	defaultContextLimit = 100_000
	thereshold          = 0.9
)

func doCompact(
	ctx context.Context,
	model agent.Model,
	msgs []agent.Message,
	evCh chan Event,
) ([]agent.Message, error) {

	if len(msgs) <= 1 {
		return nil, fmt.Errorf("must be at least one message")
	}

	var (
		preContextMsgs          []agent.Message
		onCompactionContextMsgs []agent.Message
		tailContextMsgs         []agent.Message
	)

	// keep system prompt message
	preContextMsgs = []agent.Message{}
	if systemMsg, ok := msgs[0].(*agent.SystemMessage); ok {
		preContextMsgs = append(preContextMsgs, systemMsg)
		msgs = msgs[1:]
	}

	// sepatare tail and compaction candidates
	divide := int(float32(len(msgs)) * compactionRatio)
	onCompactionContextMsgs = msgs[:divide]
	tailContextMsgs = msgs[divide:]

	compactionRequest := agent.StringifyConversation(onCompactionContextMsgs)
	compactionContext := []agent.Message{
		agent.NewSystemMessage(prompt.Compaction()),
		agent.NewUserMessage(compactionRequest),
	}

	// call llm
	completion, err := model.Complete(ctx, nil, compactionContext)
	if err != nil {
		return nil, err
	}

	// assemble new context
	summary := prompt.SummaryExplanation(completion.Content)
	tailContextMsgs = roleSafePushback(tailContextMsgs, agent.NewUserMessage(summary))
	compactedContext := append(preContextMsgs, tailContextMsgs...)

	evCh <- NewCompactionEvent(
		completion.Content,
		compactedContext,
	)

	return compactedContext, nil
}

func roleSafePushback(
	messages []agent.Message,
	newFirstMessage agent.Message,
) []agent.Message {

	firstMessage := messages[0]

	if firstMessage.Role() == newFirstMessage.Role() {

		prevFirstContent := firstMessage.Content()
		newFirstContent := newFirstMessage.Content()

		contentUnion := append(prevFirstContent, newFirstContent...)
		firstMessage.SetContent(contentUnion)

		return messages
	}

	newStack := []agent.Message{}
	newStack = append(newStack, newFirstMessage)
	newStack = append(newStack, messages...)
	return newStack
}

func shouldCompact(
	inputTokens,
	outputTokens,
	contextLimit int64,
) bool {
	if contextLimit <= 0 {
		contextLimit = defaultContextLimit
	}
	return (inputTokens + outputTokens) >= int64(float64(contextLimit)*thereshold)
}
