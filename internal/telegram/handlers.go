package telegram

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"context"
	"encoding/json"
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

	// Convert message to text format
	messageText := b.formatMessage(message)

	// Create event reader for handling agent events
	eventReader := runtime.EventReader{
		OnCompaction: func(agentID agent.ID, sessionID session.ID, reason string) {
			if _, err := b.SendMessage(message.From.ID, fmt.Sprintf("⚠️ Session compacted: %s", reason), 0); err != nil {
				slog.Warn("failed to send compaction notification", "error", err)
			}
		},
		OnComplete: func(agentID agent.ID, sessionID session.ID, completion *agent.Completion) {

			// Format tool calls
			toolRepr, err := b.formatToolCalls(completion.ToolCalls, message)
			if err != nil {
				slog.Warn("failed to format tool calls", "error", err)
			}

			// Combine tool representation and completion text
			fullResponse := toolRepr + "\n" + completion.Content

			// Send response
			if _, err := b.SendMessage(message.From.ID, fullResponse, 0); err != nil {
				slog.Warn("failed to send completion", "error", err)
			}
		},
	}

	// Process message through chat service
	err := b.chatSvc.Chat(
		ctx,
		b.agentID,
		b.sessionID,
		messageText,
		eventReader,
		b.tools,
	)

	if err != nil {
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

// formatMessage converts a Telegram message to a standardized text format
func (b *Bot) formatMessage(msg *tgbotapi.Message) string {
	var sb strings.Builder

	// Basic message info
	fmt.Fprintf(&sb, "## Telegram Message\n")
	fmt.Fprintf(&sb, "Time: %s\n", msg.Time().Format("15:04 02.01.06"))
	fmt.Fprintf(&sb, "Chat ID: %d\n", msg.From.ID)
	fmt.Fprintf(&sb, "From: %s\n", msg.From.FirstName)

	// Sticker information
	if msg.Sticker != nil && msg.Sticker.Emoji != "" {
		fmt.Fprintf(&sb, "Sticker: %s\n", msg.Sticker.Emoji)
	}

	// Message text
	if msg.Text != "" {
		sb.WriteString(msg.Text)
	}

	return sb.String()
}

// handleCommand processes bot commands
func (b *Bot) handleCommand(ctx context.Context, update tgbotapi.Update) error {
	command := update.Message.Command()

	switch command {
	case "start":
		_, err := b.SendMessage(update.Message.From.ID, "Hello! I'm your assistant bot.", 0)
		return err
	default:
		_, err := b.SendMessage(update.Message.From.ID, fmt.Sprintf("unknown command: %s", command), 0)
		return err
	}
}

// formatToolCalls creates a human-readable representation of tool calls
func (b *Bot) formatToolCalls(calls []*agent.ToolCall, msg *tgbotapi.Message) (string, error) {
	if len(calls) == 0 {
		return "", nil
	}

	var sb strings.Builder

	for _, call := range calls {
		switch call.ToolName {
		case "fetch":
			var args struct {
				URL string `json:"url"`
			}
			if err := unmarshalArgs(call.Arguments, &args); err != nil {
				return "", fmt.Errorf("failed to unmarshal fetch args: %w", err)
			}
			fmt.Fprintf(&sb, "🔍 Fetching: %s\n", args.URL)

		case "call_agent":
			var args struct {
				AgentName string `json:"name"`
			}
			if err := unmarshalArgs(call.Arguments, &args); err != nil {
				return "", fmt.Errorf("failed to unmarshal call_agent args: %w", err)
			}
			fmt.Fprintf(&sb, "🤖 Calling agent: %s\n", args.AgentName)

		case "web_search":
			var args struct {
				Query string `json:"query"`
			}
			if err := unmarshalArgs(call.Arguments, &args); err != nil {
				return "", fmt.Errorf("failed to unmarshal web_search args: %w", err)
			}
			fmt.Fprintf(&sb, "🔎 Searching: %s\n", args.Query)

		case "send_message":
			var args struct {
				ChatID int64 `json:"chat_id"`
			}
			if err := unmarshalArgs(call.Arguments, &args); err != nil {
				return "", fmt.Errorf("failed to unmarshal send_message args: %w", err)
			}
			if args.ChatID != msg.Chat.ID {
				chatInfo, err := b.api.GetChat(tgbotapi.ChatInfoConfig{
					ChatConfig: tgbotapi.ChatConfig{
						ChatID: args.ChatID,
					},
				})
				if err != nil {
					return "", fmt.Errorf("failed to get chat info: %w", err)
				}
				fmt.Fprintf(&sb, "💬 Messaging %s\n", chatInfo.FirstName)
			}

		case "create_task":
			var args struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Recipients  string `json:"recipients"`
				Reglament   string `json:"reglament"`
				Oneshot     bool   `json:"oneshot"`
			}
			if err := unmarshalArgs(call.Arguments, &args); err != nil {
				return "", fmt.Errorf("failed to unmarshal create_task args: %w", err)
			}
			fmt.Fprintf(&sb, "📝 Creating task: %s\n", args.Name)
			if args.Oneshot {
				sb.WriteString("(one-time)")
			}

		default:
			fmt.Fprintf(&sb, "🔧 Using tool: %s\n", call.ToolName)
		}
	}

	return sb.String(), nil
}

// unmarshalArgs is a helper to unmarshal tool arguments with error handling
func unmarshalArgs(data agent.ToolArguments, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal arguments: %w", err)
	}
	return nil
}
