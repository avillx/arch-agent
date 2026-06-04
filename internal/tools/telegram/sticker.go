package tgtools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/telegram"
	"arch-agent/internal/tools"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
)

var _ runtime.PerAgentInstructed = (*SendStickerTool)(nil)

// SendSticker tool
type SendStickerTool struct {
	orchestrator *telegram.BotOrchestrator
}

func NewSendStickerTool(o *telegram.BotOrchestrator) *SendStickerTool {
	return &SendStickerTool{
		orchestrator: o,
	}
}

func (t *SendStickerTool) Instruction() string {
	return `Stickers:
- Use stickers for immersive, expressive chatting.
- Only send stickers after obtaining the allowed list from the user.
- Send them when it genuinely fits the mood or context — not forced.
- It feels natural when: reacting emotionally, celebrating, sympathizing, or adding humor.`
}

func (t *SendStickerTool) AgentInstruction(agt agent.Agent) string {
	bot, err := t.orchestrator.Get(agt.ID())
	if err != nil {
		slog.Error("has no bot for agent", "error", ErrNoAcc, "agent", agt.ID())
		return ""
	}

	allowedStickers := strings.Join(slices.Collect(maps.Keys(bot.Stickers)), ", ")
	return fmt.Sprintf("- allowed stickers: %s", allowedStickers)
}

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

	agentID := tools.MustAgentID(ctx)

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
