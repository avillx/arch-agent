package tools

import (
	"arch-agent/internal/app/types"
	"arch-agent/internal/infra/llm"
	"arch-agent/internal/infra/telegram"
	"maps"
	"slices"
)

func SendMessage(b *telegram.Bot) llm.Tool {
	return llm.Tool{
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
		CallRsolver: llm.WrapArgumentedCallResolver(
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
func SendSticker(b *telegram.Bot) llm.Tool {
	return llm.Tool{
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
		CallRsolver: llm.WrapArgumentedCallResolver(
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
