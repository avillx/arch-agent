package externaltools

import (
	"arch-agent/internal/domain/types"
	"arch-agent/internal/infra/telegram"
	"maps"
	"slices"
)

func TelegramTS(b *telegram.Bot) tools.Server {
	return tools.NewToolServer(
		"telegram",
		`<telegram>
For chatting in Telegram. Act like a real human — casual, organic, imperfect.

Message count per response:
- Default: 1–3 messages
- Simple reply or reaction: 1 message
- Explaining something or telling a story: up to 4 messages max
- Never send more than 4 messages in a row without user reply
- Send stickers when it feels natural, do not send it every turn, only when you want to express something.

How to split messages:
- Each message = one thought or one beat
- Split on natural pauses, not on sentence ends
- Don't split if it feels forced — one message is fine

Organic tone:
- Don't be exhaustive. Humans don't cover everything in one go.
- Leave room for the conversation to continue.
- Match conversation energy — short input → short response. 
- You chatting now, use send_sticker for casual human conversation.
  
User never sees your raw output only sended messages.
Never repeat previous response structure (message count, sentence structure and thought that you can provide).
<telegram>`,
		[]tools.Tool{
			SendMessage(b),
			SendSticker(b),
		},
	)
}

func SendMessage(b *telegram.Bot) tools.Tool {
	return tools.Tool{
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
		CallRsolver: tools.WrapArgumentedCallResolver(
			func(args struct {
				ChatID int64  `json:"chat_id"`
				Text   string `json:"text"`
			}) (string, error) {
				if err := b.SendMessage(args.ChatID, args.Text, 0); err != nil {
					return "message is not sended", err
				}
				return "message sended", nil
			}),
	}
}
func SendSticker(b *telegram.Bot) tools.Tool {
	return tools.Tool{
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
					Name:     "emoji",
					Required: true,
					Type:     types.TypeString,
					Enum:     slices.Collect(maps.Keys(b.Stickers)),
				},
			},
		},
		CallRsolver: tools.WrapArgumentedCallResolver(
			func(
				args struct {
					ChatID int64  `json:"chat_id"`
					Emoji  string `json:"emoji"`
				},
			) (string, error) {
				if err := b.SendSticker(args.ChatID, args.Emoji); err != nil {
					return "sticker is not sended", err
				}
				return "sticker sended", nil
			}),
	}
}
