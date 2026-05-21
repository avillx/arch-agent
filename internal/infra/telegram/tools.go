package telegram

import (
	service "arch-agent/internal/app"
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/tool"
	"context"
	"errors"
)

// send message
type SendMessageTool struct {
	orchestrator *BotOrchestrator
}

func NewSendMessageTool(o *BotOrchestrator) *SendMessageTool {
	return &SendMessageTool{
		orchestrator: o,
	}
}

func (t *SendMessageTool) Name() string {
	return "send_message"
}

func (t *SendMessageTool) Description() string {
	return "send messages in chat"
}

func (t *SendMessageTool) Schema() []tool.ToolProperty {
	return []tool.ToolProperty{
		{
			Name:        "chat_id",
			Required:    true,
			Type:        tool.TypeNumber,
			Description: "chat id that message will be sended",
		},
		{
			Name:        "text",
			Required:    true,
			Type:        tool.TypeString,
			Description: "text content of your message",
		},
	}
}

func (t *SendMessageTool) Call(ctx context.Context, rawArgs tool.ToolArguments) (string, error) {
	args, err := service.UnwrapArgs[struct {
		ChatID int64  `json:"chat_id"`
		Text   string `json:"text"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	agentID := service.MustAgentID(ctx)

	bot, err := t.orchestrator.Get(agent.ID(agentID))
	if err != nil {
		return "", errors.Join(err, ErrNoAcc)
	}
	if _, err := bot.SendMessage(args.ChatID, args.Text, 0); err != nil {
		return "message is not sended", err
	}
	return "message sended", nil
}

// SendSticker tool
type SendStickerTool struct {
	orchestrator *BotOrchestrator
}

func NewSendStickerTool(o *BotOrchestrator) *SendStickerTool {
	return &SendStickerTool{
		orchestrator: o,
	}
}

func (t *SendStickerTool) Name() string {
	return "send_sticker"
}

func (t *SendStickerTool) Description() string {
	return "send sticker in chat"
}

func (t *SendStickerTool) Schema() []tool.ToolProperty {
	return []tool.ToolProperty{
		{
			Name:        "chat_id",
			Required:    true,
			Type:        tool.TypeNumber,
			Description: "chat id that sticker will be sended",
		},
		{
			Name:        "emoji",
			Required:    true,
			Type:        tool.TypeString,
			Description: "sticker emoji, never use not allowed emojis, only from enum. Only one emoji",
		},
	}
}

func (t *SendStickerTool) Call(ctx context.Context, rawArgs tool.ToolArguments) (string, error) {
	args, err := service.UnwrapArgs[struct {
		ChatID int64  `json:"chat_id"`
		Emoji  string `json:"emoji"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	agentID := service.MustAgentID(ctx)

	bot, err := t.orchestrator.Get(agent.ID(agentID))
	if err != nil {
		return "", errors.Join(err, ErrNoAcc)
	}

	if err := bot.SendSticker(args.ChatID, args.Emoji); err != nil {
		return "sticker is not sended", err
	}
	return "sticker sended", nil
}

var ErrNoAcc = errors.New("You have no telegram bot account")
