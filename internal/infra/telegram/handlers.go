package telegram

import (
	"arch-agent/internal/app/types"
	"arch-agent/internal/infra/llm"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) Tools() []llm.Tool {
	sendMessage := llm.Tool{
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

	sendSticker := llm.Tool{
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
					Enum:     slices.Collect(maps.Keys(b.stickers)),
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

	return slices.Concat([]llm.Tool{sendMessage, sendSticker})
}

// handlers
func (b *Bot) handleMessage(message *tgbotapi.Message) error {

	const OriginContext = `This is message recived from telegram. 
	You should act as human, answer organic with provided capabilities (stickers, messages etc...). 
	When you write \\n\\n this is diffirent message. 
	You should divide text on a several messages for organic and natural dialogue in messager.
	Never repeat previus answer structure
	At first it's chatting.
	User never see your output, for communicate you should use send_message, send_stciker or other tools
	`

	stopAction := b.SetChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	defer stopAction()

	contentRecivier := func(ctx context.Context, content string) error {

		fmt.Print("### Output: ", content)
		return nil
		// var errc error
		// for _, line := range strings.Split(content, "\n\n") {
		// 	err := b.SendMessage(message.From.ID, line, 0)
		// 	errc = errors.Join(errc, err)
		// }
		// return errc
	}

	content := messageToText(message)

	return Try(3, func() error {
		return b.answerUC.Execute(
			context.Background(),
			content,
			contentRecivier,
			OriginContext)
	})
}

func Try(attmpts int, function func() error) error {
	var errc error
	for i := 0; i < attmpts; i++ {
		if err := function(); err != nil {
			errc = errors.Join(errc, err)
			continue
		}
		return errc
	}
	return errors.Join(errc, errors.New("fallback attempts budget expires"))
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
