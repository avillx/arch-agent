package telegram

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/mcp"
	"arch-agent/internal/session"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Dreamer interface {
	DreamImmidate(context.Context, agent.ID) error
}

const (
	maxMessageLength   = 4096
	sessionTimeout     = 10 * time.Minute
	chatActionInterval = 4 * time.Second
)

// BotConfig holds configuration for a Telegram bot
type BotConfig struct {
	Agent          string
	APIKey         string
	Host           string
	StickerSetName string
	Logs           bool
}

// Bot represents a Telegram bot instance
type Bot struct {
	api          *tgbotapi.BotAPI
	updateChan   tgbotapi.UpdatesChannel
	stickerMap   map[string]string
	blockedUsers []int64
	sessionSvc   *session.Service
	chatSvc      *chat.Service
	mcpSvc       *mcp.Service

	agentID agent.ID

	// session management
	sessionMu    sync.Mutex
	sessionID    session.ID
	sessionTimer *time.Timer

	// mode
	isWebhook bool

	d Dreamer
}

// NewBot creates a new Telegram bot instance
func NewBot(cfg BotConfig) (*Bot, error) {
	// Initialize bot API
	botAPI, err := tgbotapi.NewBotAPI(cfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	// Configure logging
	if !cfg.Logs {
		botAPI.Debug = false
	}

	bot := &Bot{
		api:          botAPI,
		stickerMap:   make(map[string]string),
		blockedUsers: []int64{},
		agentID:      agent.ID(cfg.Agent),
		isWebhook:    cfg.Host != "",
		tools:        []agent.Tool{},
	}

	// Load stickers if configured
	if cfg.StickerSetName != "" {
		if err := bot.loadStickers(cfg.StickerSetName); err != nil {
			slog.Warn("failed to load stickers", "error", err)
		}
	}

	// Configure update channel based on mode
	if err := bot.configureUpdates(cfg.Host); err != nil {
		return nil, fmt.Errorf("failed to configure updates: %w", err)
	}

	return bot, nil
}

// Wire connects the bot to services
func (b *Bot) Wire(sessionSvc *session.Service, chatSvc *chat.Service, d Dreamer, mcpSvc *mcp.Service) {
	b.sessionSvc = sessionSvc
	b.chatSvc = chatSvc
	b.d = d
	b.mcpSvc = mcpSvc
}

// Run starts the bot's update loop
func (b *Bot) Run(ctx context.Context) {
	defer b.cleanup()

	slog.Info("telegram bot started", "mode", b.modeString())

	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-b.updateChan:
			if !ok {
				return
			}
			go b.handleUpdate(ctx, update)
		}
	}
}

// modeString returns the current bot mode for logging
func (b *Bot) modeString() string {
	if b.isWebhook {
		return "webhook"
	}
	return "longpoll"
}

// cleanup performs cleanup operations when bot stops
func (b *Bot) cleanup() {
	if b.sessionTimer != nil {
		b.sessionTimer.Stop()
	}
	if b.isWebhook {
		b.unsetWebhook()
	}
}

// configureUpdates sets up the update channel based on configuration
func (b *Bot) configureUpdates(host string) error {
	var err error

	if host != "" {
		// Webhook mode
		b.updateChan, err = b.setupWebhook(host)
		if err != nil {
			return fmt.Errorf("webhook setup failed: %w", err)
		}
	} else {
		// Long polling mode
		b.updateChan = b.setupLongPolling()
	}

	return nil
}

// setupLongPolling configures the bot for long polling
func (b *Bot) setupLongPolling() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	return b.api.GetUpdatesChan(u)
}

// setupWebhook configures the bot for webhook mode
func (b *Bot) setupWebhook(host string) (tgbotapi.UpdatesChannel, error) {
	// Remove any existing webhook
	if _, err := b.api.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true}); err != nil {
		return nil, fmt.Errorf("failed to remove existing webhook: %w", err)
	}

	// Set up new webhook
	webhookURL := fmt.Sprintf("https://%s/bots/%s", host, b.api.Token)
	wh, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}
	_, err = b.api.Request(wh)
	if err != nil {
		return nil, fmt.Errorf("failed to set webhook: %w", err)
	}

	// Verify webhook is set
	info, err := b.api.GetWebhookInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook info: %w", err)
	}

	if !info.IsSet() {
		return nil, errors.New("webhook not set after configuration")
	}

	if info.LastErrorDate != 0 {
		return nil, errors.New("webhook has pending errors")
	}

	return b.api.ListenForWebhook("/bots/" + b.api.Token), nil
}

// unsetWebhook removes the webhook when bot shuts down
func (b *Bot) unsetWebhook() {
	if _, err := b.api.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false}); err != nil {
		slog.Warn("failed to remove webhook", "error", err)
	}
}

// loadStickers loads the sticker set into memory
func (b *Bot) loadStickers(setName string) error {
	set, err := b.api.GetStickerSet(tgbotapi.GetStickerSetConfig{Name: setName})
	if err != nil {
		return fmt.Errorf("failed to get sticker set: %w", err)
	}

	if len(set.Stickers) == 0 {
		return fmt.Errorf("sticker set %s is empty", setName)
	}

	stickerMap := make(map[string]string)
	for _, sticker := range set.Stickers {
		if sticker.Emoji != "" {
			stickerMap[sticker.Emoji] = sticker.FileID
		}
	}

	b.stickerMap = stickerMap
	return nil
}

// AllowedEmojis returns the list of available sticker emojis
func (b *Bot) AllowedEmojis() []string {
	return slices.Collect(maps.Keys(b.stickerMap))
}

// SendMessage sends a text message to a user
func (b *Bot) SendMessage(userID int64, text string, replyTo int) ([]tgbotapi.Message, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("message text cannot be empty")
	}

	// Escape text for MarkdownV2
	text = tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, text)

	// Split into chunks if needed
	chunks := b.splitMessage(text)
	var messages []tgbotapi.Message

	for i, chunk := range chunks {
		msg := tgbotapi.NewMessage(userID, chunk)
		msg.ParseMode = tgbotapi.ModeMarkdownV2

		// Only set reply for the first message
		if replyTo > 0 && i == 0 {
			msg.ReplyToMessageID = replyTo
		}

		sentMsg, err := b.api.Send(msg)
		if err != nil {
			return messages, fmt.Errorf("failed to send message %d/%d: %w", i+1, len(chunks), err)
		}

		messages = append(messages, sentMsg)
	}

	return messages, nil
}

// splitMessage splits a long message into chunks that fit Telegram's limits
func (b *Bot) splitMessage(text string) []string {
	runes := []rune(text)
	var chunks []string

	if len(runes) <= maxMessageLength {
		return []string{text}
	}

	// Calculate number of chunks needed
	chunkCount := len(runes) / maxMessageLength
	if len(runes)%maxMessageLength != 0 {
		chunkCount++
	}

	for i := 0; i < chunkCount; i++ {
		start := i * maxMessageLength
		end := start + maxMessageLength
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}

	return chunks
}

// SendSticker sends a sticker to a chat
func (b *Bot) SendSticker(chatID int64, emoji string) error {
	fileID, exists := b.stickerMap[emoji]
	if !exists {
		return fmt.Errorf("sticker with emoji %s not found", emoji)
	}

	sticker := tgbotapi.NewSticker(chatID, tgbotapi.FileID(fileID))
	_, err := b.api.Send(sticker)
	if err != nil {
		return fmt.Errorf("failed to send sticker: %w", err)
	}

	return nil
}

// SetTypingAction shows typing indicator in a chat
func (b *Bot) SetTypingAction(chatID int64) (cancel func()) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)

	go func() {
		ticker := time.NewTicker(chatActionInterval)
		defer ticker.Stop()

		sendAction := func() {
			_, err := b.api.Request(tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping))
			if err != nil {
				slog.Debug("failed to send typing action", "error", err)
			}
		}

		sendAction() // Send immediately

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sendAction()
			}
		}
	}()

	return cancelFunc
}
