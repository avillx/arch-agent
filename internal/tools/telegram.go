package tools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/telegram"
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
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

func (t *SendMessageTool) Instruction() string {
	return `Telegram chatting:
- Chat naturally, like a person in a messenger.
- Match the user's message length and energy.
- Short messages usually deserve short replies.
- Do not continue a topic that naturally ended.
- Do not ask questions unless there is a reason.
- One message is usually enough.
- Silence, acknowledgement or a brief reaction can be a complete reply.
- Do not explain more than the user asked for.`
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
	args, err := UnwrapArgs[struct {
		ChatID int64  `json:"chat_id"`
		Text   string `json:"text"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	agentID := MustAgentID(ctx)

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
	args, err := UnwrapArgs[struct {
		ChatID int64  `json:"chat_id"`
		Emoji  string `json:"emoji"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	agentID := MustAgentID(ctx)

	bot, err := t.orchestrator.Get(agent.ID(agentID))
	if err != nil {
		return "", errors.Join(err, ErrNoAcc)
	}

	if err := bot.SendSticker(args.ChatID, args.Emoji); err != nil {
		return "sticker is not sended", err
	}
	return "sticker sended", nil
}

// SendSticker tool
type GetStickersTool struct {
	orchestrator *telegram.BotOrchestrator
}

func NewGetStickersTool(o *telegram.BotOrchestrator) *GetStickersTool {
	return &GetStickersTool{
		orchestrator: o,
	}
}

func (t *GetStickersTool) Name() agent.ToolName {
	return "get_stickers"
}

func (t *GetStickersTool) Description() string {
	return "returns list of emojis, allowed for stickers"
}

func (t *GetStickersTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{}
}

func (t *GetStickersTool) Call(ctx context.Context, _ agent.ToolArguments) (string, error) {

	agentID := MustAgentID(ctx)

	bot, err := t.orchestrator.Get(agent.ID(agentID))
	if err != nil {
		return "", errors.Join(err, ErrNoAcc)
	}

	return strings.Join(slices.Collect(maps.Keys(bot.Stickers)), ", "), nil
}

var ErrNoAcc = errors.New("You have no telegram bot account")
