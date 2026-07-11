package telegram

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/mcp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func processMCPCommand(ctx context.Context, mcpSvc *mcp.Service, u tgbotapi.Update) (string, error) {
	command := strings.Fields(u.Message.Text)
	if !(len(command) > 0) {
		return "", fmt.Errorf("bad usage, text must contain command has: `%s`", u.Message.Text)
	}

	// exclude '/mcp' route, keep only command
	command = command[1:]

	if len(command) == 0 {
		list := mcpSvc.List()
		if len(list) > 0 {
			return mcpListRepr(list), nil
		}
		return "has no mcp servers", nil
	}

	const helpMessage = `availavble args:
	"/mcp" - show list of added servers
	"/mcp reload"
	"/mcp add <url> <token>"
	"/mcp add <command> <ags> <env:key=value>"
	"/mcp remove <server_name>" 
	"/mcp disconnect <server_name>"
	"/mcp help" - show this message`

	switch command[0] {
	case "reload":
		if err := mcpSvc.Reload(ctx); err != nil {
			if errs, ok := err.(interface{ Unwrap() []error }); ok {
				var sb strings.Builder
				for _, e := range errs.Unwrap() {
					sb.WriteString(e.Error())
				}
				return sb.String(), nil
			}
			return "", err
		}
	case "add":

		cfg, err := commandToGatewayConfig(command[1:])
		if err != nil {
			return "", err
		}

		newMcpID, err := mcpSvc.Connect(ctx, cfg)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("mcp %s added", newMcpID), nil

	case "disconnect":

		if err := mcpSvc.Disconnect(mcp.MCPServerID(command[1])); err != nil {
			return "", err
		}
		return fmt.Sprintf("mcp %s disconnected", command[1]), nil
	case "help":

		return helpMessage, nil
	}

	return fmt.Sprintf("unknown %s command args:\n%s", command[0], helpMessage), nil
}

func commandToGatewayConfig(commandParts []string) (mcp.ServerGatewayConfig, error) {
	var cfg mcp.ServerGatewayConfig

	if len(commandParts) == 0 {
		return cfg, fmt.Errorf("Has no arguments, need path, url or npm package")
	}

	args := []string{}
	envs := map[string]string{}
	for _, p := range commandParts {
		if env, found := strings.CutPrefix(p, "env:"); found {
			parts := strings.Split(env, "=")
			if len(parts) != 2 {
				return cfg, fmt.Errorf("bad env var format it must be `env:key=value`")
			}
			envs[parts[0]] = parts[1]
			continue
		}
		args = append(args, p)
	}

	if strings.HasPrefix(args[0], "http") {
		if _, err := url.Parse(args[0]); err == nil {
			cfg.HTTPGateway = &mcp.HTTPGatewayConfig{
				URL: args[0],
			}

			if len(args) > 1 {
				cfg.HTTPGateway.Token = args[1]
			}

			return cfg, nil
		}
	}

	cfg.CommandGateway = &mcp.CommandGatewayConfig{
		Command: args[0],
		Args:    args[1:],
		Env:     envs,
	}

	return cfg, nil
}

// formatToolCalls creates a human-readable representation of tool calls
func formatToolCalls(b *Bot, calls []*agent.ToolCall, msg *tgbotapi.Message) (string, error) {
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
		case "send_sticker":
			// do nothing
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

func mcpListRepr(servers []*mcp.MCPServer) string {
	var sb strings.Builder

	for _, s := range servers {
		fmt.Fprintf(&sb, "Server: %s\n\n", s.ID)
	}
	return sb.String()
}

// formatMessage converts a Telegram message to a standardized text format
func toMessage(b *Bot, msg *tgbotapi.Message) *agent.UserMessage {
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

	parts := []agent.ContentPart{}

	// Message text
	if msg.Text != "" {
		sb.WriteString(msg.Text)
	}

	parts = append(parts, agent.ContentPart{
		Text: sb.String(),
	})

	if len(msg.Photo) > 0 {
		photo := msg.Photo[len(msg.Photo)-1]
		contentPart, err := resolveImage(b, photo)

		if err != nil {
			slog.Error("can't resolve image", "error", err)
			contentPart = agent.ContentPart{
				Text: "some errors in reading image",
			}
		}
		parts = append(parts, contentPart)
	}

	return agent.NewUserMessage(parts)
}

func resolveImage(b *Bot, image tgbotapi.PhotoSize) (agent.ContentPart, error) {
	file, err := b.api.GetFile(tgbotapi.FileConfig{FileID: image.FileID})
	if err != nil {
		return agent.ContentPart{}, err
	}

	resp, err := http.Get(file.Link(b.api.Token))
	if err != nil {
		return agent.ContentPart{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return agent.ContentPart{}, fmt.Errorf("bad request status %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return agent.ContentPart{}, err
	}

	ct := http.DetectContentType(data)
	return agent.NewImageContent(agent.AllowedMIME(ct), data)
}
