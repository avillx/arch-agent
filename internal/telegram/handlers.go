package telegram

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"context"
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

	evReader := runtime.EventReader{
		OnComplete: func(_ agent.ID, _ session.ID, c *agent.Completion) {
			if c.Content != "" {
				b.SendMessage(message.From.ID, c.Content, message.MessageID)
			}
		},
	}

	sess, err := b.sessionService.Get(b.agentID, "session_test")
	if err != nil {
		return err
	}

	sess.AddMessages([]agent.Message{agent.NewUserMessage(messageToText(message))})

	agt, err := b.agentRepo.Get(b.agentID)
	if err != nil {
		return err
	}

	model, err := b.modelRepo.Get(agt.Model())
	if err != nil {
		return err
	}

	sink := b.agentRuntime.RunStream(
		context.TODO(),
		model,
		agt,
		agt.Tools(),
		sess,
	)

	evReader.Read(sink)

	b.sessionService.Save(b.agentID, sess)

	return nil
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
