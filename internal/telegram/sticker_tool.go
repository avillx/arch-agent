package telegram

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"errors"
)

// SendSticker tool
type SendStickerTool struct {
	bot    *Bot
	chatID int64
}

func (t *SendStickerTool) Instruction() string {
	return `## Stickers:
- Use stickers for immersive, expressive chatting.
- Send them when it genuinely fits the mood or context — not forced.
- It feels natural when: reacting emotionally, celebrating, sympathizing, or adding humor.`
}

func (t *SendStickerTool) Name() agent.ToolName {
	return "send_sticker"
}

func (t *SendStickerTool) Description() string {
	return "Send a sticker  to the current chat by emoji"
}

func (t *SendStickerTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:        "chat_id",
			Required:    true,
			Type:        agent.TypeNumber,
			Description: "Telegram chat ID to send sticker to",
		},
		{
			Name:        "emoji",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Single emoji character representing the reaction. Must be one of the allowed values from enum.",
			Enum:        t.bot.AllowedEmojis(),
		},
	}
}

func (t *SendStickerTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Emoji string `json:"emoji"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	if err := t.bot.SendSticker(t.chatID, args.Emoji); err != nil {
		return agent.NewContent("sticker sended"), err
	}
	return agent.NewContent("sticker sended"), nil
}

var ErrNoAcc = errors.New("You have no telegram bot account")
