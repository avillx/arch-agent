package telegram

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/session"
	tgtools "arch-agent/internal/telegram/telegram"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	maxMessageTextLen  = 4096
	sessionExpiresTime = 10 * time.Minute
)

type StickerMap map[string]string

type BotConfig struct {
	Agent          string
	APIKey         string
	Host           string
	StickerSetName string
	Logs           bool
}

type Bot struct {
	API           *tgbotapi.BotAPI
	updateChannel tgbotapi.UpdatesChannel
	Stickers      StickerMap
	blockedUsers  []int64
	sessionSvc    *session.Service
	chatSvc       *chat.Service
	agentID       agent.ID

	sessionTimer *time.Timer
	sessionID    session.ID

	tools []agent.Tool
}

func NewBot(cfg BotConfig) (*Bot, error) {

	// build api
	botAPI, err := tgbotapi.NewBotAPI(cfg.APIKey)
	if err != nil {
		return nil, err
	}

	if !cfg.Logs {
		botAPI.Debug = false
		tgbotapi.SetLogger(
			slog.NewLogLogger(
				slog.NewTextHandler(io.Discard, nil),
				slog.LevelError,
			),
		)
	}

	// build bot
	bot := &Bot{
		API:          botAPI,
		blockedUsers: []int64{},
		agentID:      agent.ID(cfg.Agent),
		tools:        []agent.Tool{},
	}

	bot.sessionTimer = time.AfterFunc(sessionExpiresTime, func() {
		bot.sessionID = ""
	})

	// add tools
	bot.tools = append(
		bot.tools,
		// tgtools.NewSendMessageTool(bot),
		tgtools.NewSendStickerTool(bot),
	)

	// set stickers
	if cfg.StickerSetName != "" {
		stickers, err := bot.stickerMap(cfg.StickerSetName)
		switch {
		case err != nil:
			slog.Error("bot creation", "error", err)
		default:
			bot.Stickers = stickers
		}
	}

	// config update channel
	switch {
	// config web hook is not null
	case cfg.Host != "":
		bot.updateChannel, err = createWebhookUpdateChannelFor(bot, cfg.Host)
		if err != nil {
			return nil, err
		}
		slog.Info("telegram bot started with webhook")
	default:
		bot.updateChannel = createPollingUpdateChannelFor(bot)
		slog.Info("telegram bot started with polling")
	}

	return bot, nil

}

func (b *Bot) Wire(
	sessionSvc *session.Service,
	chatSvc *chat.Service,
) {
	b.chatSvc = chatSvc
	b.sessionSvc = sessionSvc
}

func (b *Bot) SendMessage(userID int64, text string, replyMessageID int) ([]tgbotapi.Message, error) {

	text = tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, text)

	if len(text) <= 0 {

		return nil, errors.New("Empty string in message")
	}

	chunks := textChunkDivide(text)

	msgs := make([]tgbotapi.Message, len(chunks))

	for i, chunk := range chunks {

		msgConf := tgbotapi.NewMessage(userID, chunk)
		msgConf.ParseMode = tgbotapi.ModeMarkdownV2

		if replyMessageID > 0 && i < 1 {
			msgConf.ReplyToMessageID = replyMessageID
		}

		msg, err := b.API.Send(msgConf)
		if err != nil {
			return nil, err
		}

		msgs[i] = msg
	}

	return msgs, nil
}

func (b *Bot) SendSticker(chatID int64, emoji string) error {
	if fileId, ok := b.Stickers[emoji]; ok {
		return b.sendStickerByfileID(chatID, fileId)
	}
	return fmt.Errorf("sticker with %s emoji is not exist", emoji)
}

func (b *Bot) sendStickerByfileID(chatID int64, fileID string) error {

	msg := tgbotapi.NewSticker(chatID, tgbotapi.FileID(fileID))

	_, err := b.API.Send(msg)
	if err != nil {

		slog.Error("Cannot send sticker\n" + err.Error())
	}
	return err
}

func (b *Bot) SetChatAction(chatID int64, action string) context.CancelFunc {

	ctx, Cancel := context.WithTimeout(context.Background(), 60*time.Second)

	sendChatAction := func() {
		b.API.Send(tgbotapi.NewChatAction(chatID, action))
	}

	go func() {

		sendChatAction()

		ticker := time.NewTicker(4500 * time.Millisecond)
		defer ticker.Stop()

		for {

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sendChatAction()
			}

		}
	}()

	return Cancel
}

// blocking
func (b *Bot) Run(ctx context.Context) {
	defer b.unsetWebhook()

	for {
		select {
		case <-ctx.Done():
			return
		case update := <-b.updateChannel:
			go func() {
				if err := b.handleUpdate(update); err != nil {
					slog.Error("bad update processing", "error", err.Error())
				}
			}()
		}
	}
}

// deploy
func createPollingUpdateChannelFor(b *Bot) tgbotapi.UpdatesChannel {
	b.API.Debug = true // hardcoded shit
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	return b.API.GetUpdatesChan(u)
}

func createWebhookUpdateChannelFor(b *Bot, host string) (tgbotapi.UpdatesChannel, error) {
	dwh := tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true}
	if _, err := b.API.Request(dwh); err != nil {
		return nil, err
	}

	info, err := b.API.GetWebhookInfo()
	if err != nil {
		return nil, err
	}

	if !info.IsSet() {
		if err := b.setWebhook(host); err != nil {
			return nil, err
		}
	}

	return b.API.ListenForWebhook("/bots/" + b.API.Token), nil
}

func (b *Bot) setWebhook(host string) error {
	url := fmt.Sprintf("https://%s/bots/%s", host, b.API.Token)
	wh, _ := tgbotapi.NewWebhook(url)

	if _, err := b.API.Request(wh); err != nil {
		return errors.New("Webhook request have errors")
	}

	info, err := b.API.GetWebhookInfo()
	if err != nil {
		return err
	}

	if info.LastErrorDate != 0 {
		return errors.New("Webhook non zero time from error")
	}

	return nil
}

func (b *Bot) unsetWebhook() {
	info, err := b.API.GetWebhookInfo()
	if err != nil {
		slog.Error("bad webhook check", "error", err)
		return
	}
	if info.IsSet() {
		wh := tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false}
		if _, err := b.API.Request(wh); err != nil {
			slog.Error("bad webhook unsetting", "error", err)
		}
	}
}
