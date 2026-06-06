package tgtools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"errors"
)

// SendSticker tool
type SendStickerTool struct {
	bot Bot
}

func NewSendStickerTool(b Bot) *SendStickerTool {
	return &SendStickerTool{
		bot: b,
	}
}

// func (t *SendStickerTool) Instruction() string {
// 	return `Stickers:
// - Use stickers for immersive, expressive chatting.
// - Send them when it genuinely fits the mood or context — not forced.
// - It feels natural when: reacting emotionally, celebrating, sympathizing, or adding humor.`
// }

func (t *SendStickerTool) Name() agent.ToolName {
	return "send_sticker"
}

func (t *SendStickerTool) Description() string {
	return "send sticker in chat"
}

func (t *SendStickerTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "chat_id",
			Required:    true,
			Type:        agent.TypeNumber,
			Description: "chat id that sticker will be sended",
		},
		{
			Name:        "emoji",
			Required:    true,
			Type:        agent.TypeString,
			Description: "sticker emoji, never use not allowed emojis, only from enum. Only one emoji",
			Enum:        t.bot.AllowedEmojis(),
		},
	}
}

func (t *SendStickerTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		ChatID int64  `json:"chat_id"`
		Emoji  string `json:"emoji"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	if err := t.bot.SendSticker(args.ChatID, args.Emoji); err != nil {
		return "sticker is not sended", err
	}
	return "sticker sended", nil
}

var ErrNoAcc = errors.New("You have no telegram bot account")
