package telegram

import (
	service "arch-agent/internal/app"
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/types"
	"errors"
	"maps"
	"slices"
	"strings"
)

func TelegramTS(o *BotOrchestrator) *service.InternalServer {
	return service.NewInternalServer(
		"telegram",
		func(agentID agent.ID) string {

			stickers := ""
			if bot, err := o.Get(agentID); err == nil {
				stickers = strings.Join(slices.Collect(maps.Keys(bot.Stickers)), ", ")
			}

			return `<telegram>
Chat in Telegram like a real human.

Default is 1 message. Use 2 only if truly needed. 3 is the hard cap — never more.
Send a sticker only when it feels genuinely right, not every reply.

- Allowed sticker emojis:{ ` + stickers + `}  this is enum.

Short input → short reply. Leave room for the conversation to breathe.
<telegram>`
		},
		SendMessage(o),
		SendSticker(o),
	)
}

func SendMessage(o *BotOrchestrator) *service.InternalTool {
	return &service.InternalTool{
		ToolDefinition: types.ToolDefinition{
			Name:        "send_message",
			Description: "send messages in chat",
			Properties: []types.ToolProperty{
				{
					Name:        "chat_id",
					Required:    true,
					Type:        types.TypeNumber,
					Description: "chat id that message will be sended",
				},
				{
					Name:        "text",
					Required:    true,
					Type:        types.TypeString,
					Description: "text content of your message",
				},
			},
		},
		CallRsolver: service.WrapArgumentedCallResolver(
			func(
				args struct {
					ChatID int64  `json:"chat_id"`
					Text   string `json:"text"`
				},
				agentID string,
			) (string, error) {

				bot, err := o.Get(agent.ID(agentID))
				if err != nil {
					return "", errors.Join(err, ErrNoAcc)
				}
				if _, err := bot.SendMessage(args.ChatID, args.Text, 0); err != nil {
					return "message is not sended", err
				}
				return "message sended", nil
			}),
	}
}
func SendSticker(o *BotOrchestrator) *service.InternalTool {
	return &service.InternalTool{
		ToolDefinition: types.ToolDefinition{
			Name:        "send_sticker",
			Description: "send sticker in chat",
			Properties: []types.ToolProperty{
				{
					Name:        "chat_id",
					Required:    true,
					Type:        types.TypeNumber,
					Description: "chat id that sticker will be sended",
				},
				{
					Name:        "emoji",
					Required:    true,
					Type:        types.TypeString,
					Description: "sticker emoji, never use not allowed emojis, only from enum. Only one emoji",
				},
			},
		},
		CallRsolver: service.WrapArgumentedCallResolver(
			func(
				args struct {
					ChatID int64  `json:"chat_id"`
					Emoji  string `json:"emoji"`
				},
				agentID string,
			) (string, error) {
				bot, err := o.Get(agent.ID(agentID))
				if err != nil {
					return "", errors.Join(err, ErrNoAcc)
				}

				if err := bot.SendSticker(args.ChatID, args.Emoji); err != nil {
					return "sticker is not sended", err
				}
				return "sticker sended", nil
			}),
	}
}

var ErrNoAcc = errors.New("You have no telegram bot account")
