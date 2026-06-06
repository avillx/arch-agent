package telegram

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleMessage(message *tgbotapi.Message) error {

	// group ignore
	if (message.Chat.IsGroup() || message.Chat.IsSuperGroup()) && !b.API.IsMessageToMe(*message) {
		return nil
	}

	// session expiration
	switch b.sessionID {
	case "":
		sessID, err := b.sessionSvc.Create(b.agentID)
		if err != nil {
			return err
		}
		b.sessionID = sessID
	}
	b.sessionTimer.Stop()
	b.sessionTimer.Reset(sessionExpiresTime)

	var errc error

	b.API.Send(tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping))

	evReader := runtime.EventReader{
		OnCompaction: func(i1 agent.ID, i2 session.ID, s string) {
			if _, msgErr := b.SendMessage(message.From.ID, "⚠️ session has compacted", 0); msgErr != nil {
				errc = errors.Join(errc, msgErr)
			}
		},
		OnComplete: func(i1 agent.ID, i2 session.ID, c *agent.Completion) {

			b.API.Send(tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping))

			toolCallReprs, err := toolCallRepr(c.ToolCalls, message, b)
			if err != nil {
				errc = errors.Join(errc, err)
			}

			msgContent := fmt.Sprintf("%s\n%s", toolCallReprs, c.Content)

			if _, msgErr := b.SendMessage(message.From.ID, msgContent, 0); msgErr != nil {
				errc = errors.Join(errc, msgErr)
			}
		},
	}

	if err := b.chatSvc.Chat(
		context.Background(),
		b.agentID,
		b.sessionID,
		messageToText(message),
		evReader,
		b.tools,
	); err != nil {
		errc = errors.Join(errc, err)
		if _, msgErr := b.SendMessage(message.From.ID, "❗️ internal error", 0); msgErr != nil {
			errc = errors.Join(errc, msgErr)
		}
	}

	return errc
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
	case update.Message != nil:
		err = b.handleMessage(update.Message)
	}

	return err
}
