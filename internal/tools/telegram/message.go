package tgtools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/telegram"
	"arch-agent/internal/tools"
	"context"
	"errors"
)

// send message
type SendMessageTool struct {
	orchestrator *telegram.BotOrchestrator
}

func NewSendMessageTool(o *telegram.BotOrchestrator) *SendMessageTool {
	return &SendMessageTool{
		orchestrator: o,
	}
}

// func (t *SendMessageTool) Instruction() string {
// 	return `Telegram chatting:
// - Chat naturally, like a person in a messenger.
// - Match the user's message length and energy.
// - Short messages usually deserve short replies.
// - Do not continue a topic that naturally ended.
// - Do not ask questions unless there is a reason.
// - One message is usually enough.
// - acknowledgement or a brief reaction can be a complete reply.
// - Do not explain more than the user asked for.`
// }

func (t *SendMessageTool) Name() agent.ToolName {
	return "send_message"
}

func (t *SendMessageTool) Description() string {
	return "send messages in chat"
}

func (t *SendMessageTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "chat_id",
			Required:    true,
			Type:        agent.TypeNumber,
			Description: "chat id that message will be sended",
		},
		{
			Name:        "text",
			Required:    true,
			Type:        agent.TypeString,
			Description: "text content of your message",
		},
	}
}

func (t *SendMessageTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		ChatID int64  `json:"chat_id"`
		Text   string `json:"text"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	agentID := tools.MustAgentID(ctx)

	bot, err := t.orchestrator.Get(agent.ID(agentID))
	if err != nil {
		return "", errors.Join(err, ErrNoAcc)
	}
	if _, err := bot.SendMessage(args.ChatID, args.Text, 0); err != nil {
		return "message is not sended", err
	}
	return "message sended", nil
}
