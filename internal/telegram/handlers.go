package telegram

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleUpdate processes incoming Telegram updates
func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	// Recover from panics to prevent crash
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in update handler", "error", r)
		}
	}()

	// Ignore updates without chat information
	chat := update.FromChat()
	if chat == nil {
		slog.Debug("update without chat information", "update_id", update.UpdateID)
		return
	}

	// Check if user is blocked
	if b.isUserBlocked(chat.ID) {
		if _, err := b.SendMessage(chat.ID, "You are blocked from using this bot.", 0); err != nil {
			slog.Warn("failed to send blocked message", "error", err)
		}
		return
	}

	// Route update to appropriate handler
	var err error
	switch {
	case update.Message != nil && update.Message.IsCommand():
		err = b.handleCommand(ctx, update)
	case update.Message != nil:
		err = b.handleMessage(ctx, update.Message)
	default:
		slog.Debug("unsupported update type", "update_id", update.UpdateID)
	}

	if err != nil {
		if _, sendErr := b.SendMessage(chat.ID, "internal error occured", 0); sendErr != nil {
			err = errors.Join(err, sendErr)
		}
		slog.Error("failed to handle update", "error", err, "update_id", update.UpdateID)
	}
}

// isUserBlocked checks if a user is in the blocked list
func (b *Bot) isUserBlocked(userID int64) bool {
	for _, blockedID := range b.blockedUsers {
		if blockedID == userID {
			return true
		}
	}
	return false
}

// handleMessage processes text messages
func (b *Bot) handleMessage(ctx context.Context, message *tgbotapi.Message) error {
	// Ignore messages not addressed to the bot in groups
	if (message.Chat.IsGroup() || message.Chat.IsSuperGroup()) && !b.api.IsMessageToMe(*message) {
		return nil
	}

	// Ensure we have a valid session
	if err := b.ensureSession(); err != nil {
		return fmt.Errorf("session error: %w", err)
	}

	// Show typing indicator
	typingCancel := b.SetTypingAction(message.Chat.ID)
	defer typingCancel()

	// Create event reader for handling agent events
	eventReader := runtime.EventReader{
		OnCompaction: func(agentID agent.ID, sessionID session.ID, reason string) {
			if _, err := b.SendMessage(message.From.ID, fmt.Sprintf("⚠️ Session compacted: %s", reason), 0); err != nil {
				slog.Warn("failed to send compaction notification", "error", err)
			}
		},
		OnComplete: func(agentID agent.ID, sessionID session.ID, completion *agent.Completion) {

			// Format tool calls
			toolRepr, err := formatToolCalls(b, completion.ToolCalls, message)
			if err != nil {
				slog.Warn("failed to format tool calls", "error", err)
			}

			// send response
			msgs := strings.Split(completion.Content, "\n\n")
			for i, m := range msgs {
				if i == 0 {
					m += toolRepr
				}

				if m != "" {
					if _, err := b.SendMessage(message.From.ID, m, 0); err != nil {
						slog.Warn("failed to send completion", "error", err)
					}
				}
			}
		},
	}

	// Process message through chat service
	err := b.chatSvc.Chat(
		ctx,
		chat.Request{
			AgentID:       b.agentID,
			SessionID:     b.sessionID,
			UserMessage:   toMessage(b, message),
			Reader:        eventReader,
			ProvidedTools: b.tools,
			Logging:       true,
		},
	)

	if err != nil && !errors.Is(err, context.Canceled) {
		if _, sendErr := b.SendMessage(message.From.ID, "❗️ An error occurred while processing your message.", 0); sendErr != nil {
			return fmt.Errorf("chat error: %w, send error: %v", err, sendErr)
		}
		return fmt.Errorf("chat processing failed: %w", err)
	}

	return nil
}

// ensureSession ensures we have a valid session, creating one if needed
func (b *Bot) ensureSession() error {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	// Reset timer if session exists
	if b.sessionID != "" {
		if b.sessionTimer != nil {
			b.sessionTimer.Reset(sessionTimeout)
		}
		return nil
	}

	// Create new session
	sessionID, err := b.sessionSvc.Create(b.agentID)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	b.sessionID = sessionID

	// Start session timer
	if b.sessionTimer != nil {
		b.sessionTimer.Reset(sessionTimeout)
	} else {
		b.sessionTimer = time.AfterFunc(sessionTimeout, func() {
			b.sessionMu.Lock()
			defer b.sessionMu.Unlock()
			b.sessionID = ""
		})
	}

	return nil
}

// handleCommand processes bot commands
func (b *Bot) handleCommand(ctx context.Context, update tgbotapi.Update) error {
	command := update.Message.Command()

	switch command {
	case "start":
		_, err := b.SendMessage(update.Message.From.ID, "Hello! I'm your assistant bot.", 0)
		return err
	case "dream":

		if _, err := b.SendMessage(update.Message.From.ID, "Dreaming stated", 0); err != nil {
			return err
		}

		if err := b.d.DreamImmidate(ctx, b.agentID); err != nil {
			return err
		}

		_, err := b.SendMessage(update.Message.From.ID, "Dreaming finished", 0)
		return err
	case "mcp":
		res, err := processMCPCommand(ctx, b.mcpSvc, update)
		if err != nil {
			res += err.Error()
			err = fmt.Errorf("user mcp interactions occured %w", err)
		}
		_, sendErr := b.SendMessage(update.Message.From.ID, res, 0)
		return errors.Join(err, sendErr)
	default:
		_, err := b.SendMessage(update.Message.From.ID, fmt.Sprintf("unknown command: %s", command), 0)
		return err
	}
}
