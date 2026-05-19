package telegram

import (
	"arch-agent/internal/domain/agent"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handlers
func (b *Bot) handleMessage(message *tgbotapi.Message) error {

	if (message.Chat.IsGroup() || message.Chat.IsSuperGroup()) && !b.API.IsMessageToMe(*message) {
		return nil
	}

	stopAction := b.SetChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	defer stopAction()

	if b.app == nil {
		slog.Error("not wired uc")
		return nil
	}

	return b.app.LiveChatSvc.Chat(
		context.Background(),
		agent.ID(b.agent),
		messageToText(message),
		func(result *agent.ReasonResult) {

			if result.Content != "" {
				b.SendMessage(message.From.ID, result.Content, message.MessageID)
			}

		},
	)
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
		if _, err := b.SendMessage(update.Message.From.ID, "Hello", 0); err != nil {
			return err
		}
		return nil
	}

	// cleanUp := func(update tgbotapi.Update) error {
	// 	// really bad // b.adressant.MemoryModule.CleanContext()
	// 	return b.SendMessage(update.Message.From.ID, "memoriztion started", 0)
	// }

	commnadMap := map[string]CommandHandler{
		// "cleanup": cleanUp,
		"start": startCommand,
	}

	if handler, ok := commnadMap[update.Message.Command()]; ok {
		return handler(update)
	}

	return fmt.Errorf("command not found")
}

func (b *Bot) handleUpdate(update tgbotapi.Update) error {
	var err error

	chat := update.FromChat()
	if chat == nil {
		slog.Warn("unreadable chat in event", "update", update)
		return nil
	}
	// if user is blocked
	if slices.Contains(b.blockedUsers, chat.ID) {
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
