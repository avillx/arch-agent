package tgtools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot interface {
	SendMessage(userID int64, text string, replyMessageID int) ([]tgbotapi.Message, error)
	SendSticker(chatID int64, emoji string) error
	AllowedEmojis() []string
}

// send message
type SendMessageTool struct {
	bot Bot
}

func NewSendMessageTool(b Bot) *SendMessageTool {
	return &SendMessageTool{
		bot: b,
	}
}

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

	if _, err := t.bot.SendMessage(args.ChatID, args.Text, 0); err != nil {
		return "message is not sended", err
	}
	return "message sended", nil
}
