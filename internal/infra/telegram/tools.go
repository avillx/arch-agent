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
			if bot := o.Get(agentID); bot != nil {
				stickers = strings.Join(slices.Collect(maps.Keys(bot.Stickers)), ", ")
			}

			return `<telegram>
For chatting in Telegram. Act like a real human — casual, organic, imperfect.

Stickers:
- For send_sticker pick one emoji.
- Allowed sticker emojis:{ ` + stickers + `}  this is enum.
- For not send_sticker use emojis by other policy.
- Send stickers when it feels natural, do not send it every turn, only when you want to express something.

Message count per response:
- Default: 1–3 messages
- Simple reply or reaction: 1 message
- Explaining something or telling a story: up to 4 messages max
- Never send more than 4 messages in a row without user reply

How to split messages:
- Each message = one thought or one beat
- Split on natural pauses, not on sentence ends
- Don't split if it feels forced — one message is fine

Organic tone:
- Don't be exhaustive. Humans don't cover everything in one go.
- Leave room for the conversation to continue.
- Match conversation energy — short input → short response. 
- You chatting now, use send_sticker for casual human conversation.
  
Never repeat previous response structure (message count, sentence structure and thought that you can provide).
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

				bot := o.Get(agent.ID(agentID))
				if bot == nil {
					return "", errors.New("You have no telegram bot account")
				}
				if err := bot.SendMessage(args.ChatID, args.Text, 0); err != nil {
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
					Description: "sticker emoji, never use not allowed emojis, only from enum",
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
				bot := o.Get(agent.ID(agentID))
				if bot == nil {
					return "", errors.New("You have no telegram bot account")
				}

				if err := bot.SendSticker(args.ChatID, args.Emoji); err != nil {
					return "sticker is not sended", err
				}
				return "sticker sended", nil
			}),
	}
}
