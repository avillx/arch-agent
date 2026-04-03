package telegram

import (
	tools "arch-agent/internal/app/toolexecutor"
	"arch-agent/internal/infra/llm"
	"context"
	"fmt"
	"maps"
	"slices"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) createToolCallRecivier() *llm.ToolCallRecivier {
	sendMessage := llm.Tool{
		ToolDefinition: tools.ToolDefinition{
			Name:       "send_message",
			ReasonOnce: true,
			Strict:     true,
			Schema: tools.Schema{
				Description: "send messages in chat",
				Properties: []tools.ToolProperty{
					{
						Name:        "chat_id",
						Required:    true,
						Type:        tools.TypeNumber,
						Description: "chat id that message will be sended",
					},
					{
						Name:        "text",
						Required:    true,
						Type:        tools.TypeString,
						Description: "text content of your message",
					},
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

	sendSticker := llm.Tool{
		ToolDefinition: tools.ToolDefinition{
			Name:       "send_sticker",
			Strict:     true,
			ReasonOnce: true,
			Schema: tools.Schema{
				Description: "send sticker in chat",
				Properties: []tools.ToolProperty{
					{
						Name:        "chat_id",
						Required:    true,
						Type:        tools.TypeNumber,
						Description: "chat id that sticker will be sended",
					},
					{
						Name:     "emoji",
						Required: true,
						Type:     tools.TypeString,
						Enum:     slices.Collect(maps.Keys(b.stickers)),
					},
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

	return llm.NewToolCallRecivier([]llm.Tool{
		sendMessage,
		sendSticker,
	})
}

// handlers
func (b *Bot) handleMessage(message *tgbotapi.Message) error {

	const OriginContext = "This is message recived from telegram. You can answer organic with provided capabilities (stickers, messages etc...)"

	stopAction := b.SetChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	defer stopAction()

	contentRecivier := func(ctx context.Context, content string) error {
		return b.SendMessage(message.From.ID, content, 0)
	}

	content := messageToText(message)
	return b.answerUC.Execute(context.Background(), content, contentRecivier, OriginContext, b.createToolCallRecivier())
}

func (b *Bot) handleCommand(update tgbotapi.Update) error {
	type CommandHandler func(update tgbotapi.Update) error

	startCommand := func(update tgbotapi.Update) error {
		return b.SendMessage(update.Message.From.ID, "Hello", 0)
	}

	cleanUp := func(update tgbotapi.Update) error {
		// really bad // b.adressant.MemoryModule.CleanContext()
		return b.SendMessage(update.Message.From.ID, "memoriztion started", 0)
	}

	commnadMap := map[string]CommandHandler{
		"cleanup": cleanUp,
		"start":   startCommand,
	}

	if handler, ok := commnadMap[update.Message.Command()]; ok {
		return handler(update)
	}

	return fmt.Errorf("command not found")
}

func (b *Bot) handleUpdate(update tgbotapi.Update) error {
	var err error

	// if user is blocked
	if slices.Contains(b.blockedUsers, update.FromChat().ID) {
		b.SendMessage(update.FromChat().ID, "You in 44lab blacklist", 0)
		return nil
	}

	// route update
	switch {
	case update.Message != nil && update.Message.Command() != "":
		err = b.handleCommand(update)
	case update.Message != nil && update.Message.Text != "":
		err = b.handleMessage(update.Message)
	}

	return err
}
