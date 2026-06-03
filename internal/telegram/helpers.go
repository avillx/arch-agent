package telegram

import (
	"arch-agent/internal/agent"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) getStickerSet(name string) ([]tgbotapi.Sticker, error) {
	set, err := b.API.GetStickerSet(tgbotapi.GetStickerSetConfig{
		Name: name,
	})
	if err != nil {
		return []tgbotapi.Sticker{}, err
	}
	return set.Stickers, nil
}

func (b *Bot) getStickerMap(stickerSetName string) (StickerMap, error) {

	set, err := b.getStickerSet(stickerSetName)
	if err != nil || len(set) == 0 {
		return nil, fmt.Errorf("Sticker set not found %s", stickerSetName)
	}

	stickerMap := StickerMap{}
	for _, s := range set {
		stickerMap[s.Emoji] = s.FileID
	}

	return stickerMap, nil
}

func textChunkDivide(t string) []string {

	var runes = []rune(t)
	var chunks = []string{}
	var textLen = len(runes)

	if textLen < maxMessageTextLen {

		return []string{t}
	}

	if textLen >= maxMessageTextLen {

		chunksCount := textLen / maxMessageTextLen
		for range chunksCount {

			newChunk := runes[:maxMessageTextLen]
			chunks = append(chunks, string(newChunk))

			runes = runes[maxMessageTextLen:]
		}
	}

	return chunks
}

func messageToText(msg *tgbotapi.Message) string {
	var sb strings.Builder

	sb.WriteString("## tg message\n")
	fmt.Fprintf(&sb, "time: %s\n", msg.Time().Format("15:04 02.01.06"))
	fmt.Fprintf(&sb, "chat: %d\n", msg.From.ID)
	fmt.Fprintf(&sb, "from: %s\n", msg.From.FirstName)

	if msg.Sticker != nil {
		fmt.Fprintf(&sb, "sticker: %s\n", msg.Sticker.Emoji)
	}

	if msg.Text != "" {
		sb.WriteString(msg.Text)
	}

	// if msg.Document

	return sb.String()
}

func toolCallRepr(toolCalls []*agent.ToolCall, msg *tgbotapi.Message, b *Bot) (string, error) {
	var errc error
	var sb strings.Builder
	for _, tc := range toolCalls {
		sb.WriteString("⚒️ ")
		switch tc.ToolName {
		case "fetch":
			var args struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}
			fmt.Fprintf(&sb, "fetching %s", args.URL)

		case "call_agent":
			var args struct {
				AgentName string `json:"name"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}
			fmt.Fprintf(&sb, "call %s", args.AgentName)

		case "toggle_task":
			var args struct {
				AgentName string `json:"name"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}
			fmt.Fprintf(&sb, "call %s", args.AgentName)

		case "create_task":
			var args struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Recipients  string `json:"recipients"`
				Reglament   string `json:"reglament"`
				Oneshot     bool   `json:"oneshot"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}

			fmt.Fprintf(&sb, "> created task %s\n", args.Name)
			fmt.Fprintf(&sb, "reglament %s", args.Reglament)
			if args.Oneshot {
				sb.WriteString(" once")
			}
			fmt.Fprintf(&sb, "\nfor: %s\n", args.Recipients)
			fmt.Fprintf(&sb, "%s", args.Description)

		case "send_message":
			var args struct {
				ChatID int64 `json:"chat_id"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}
			if args.ChatID != msg.Chat.ID {
				chatInfo, err := b.API.GetChat(tgbotapi.ChatInfoConfig{
					ChatConfig: tgbotapi.ChatConfig{
						ChatID: args.ChatID,
					},
				})
				if err != nil {
					errc = errors.Join(errc, err)
					break
				}

				fmt.Fprintf(&sb, "> text to %s", chatInfo.FirstName)
			}

		case "web_search":
			var args struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}

			fmt.Fprintf(&sb, "> searching %s", args.Query)

		case "search_files":
			var args struct {
				Root  string `json:"root"`
				Query string `json:"query"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}
			fmt.Fprintf(&sb, "search %q in %s", args.Query, args.Root)

		case "delete_file":
			var args struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}
			fmt.Fprintf(&sb, "delete %s", args.Path)

		case "move_file":
			var args struct {
				Src string `json:"src"`
				Dst string `json:"dst"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}
			fmt.Fprintf(&sb, "move %s → %s", args.Src, args.Dst)

		case "edit_file":
			var args struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}
			fmt.Fprintf(&sb, "edit %s", args.Path)

		case "write_file":
			var args struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}
			fmt.Fprintf(&sb, "write %s", args.Path)

		case "list_dir":
			var args struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal(tc.Arguments, &args); err != nil {
				errc = errors.Join(errc, err)
				break
			}
			fmt.Fprintf(&sb, "ls %s", args.Path)
		}
		sb.WriteString("\n")
	}

	return sb.String(), errc
}
