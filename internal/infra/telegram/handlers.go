package telegram

import (
	"context"
	"errors"
	"fmt"
	"slices"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handlers
func (b *Bot) handleMessage(message *tgbotapi.Message) error {

	const OriginContext = `At first it's chatting telegram.
	You should act as human, answer organic with provided capabilities (stickers, messages etc...). 
	You should divide text on a several messages for organic and natural dialogue in messager.
	Never repeat previus answer structure 
	You already text a way way faster than user, if it is not answer just await.
	User never see your output, for communicate you should use send_message, send_stciker or other tools
	Sometimes one message is organic.
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
