package telegram

import (
	service "arch-agent/internal/app"
	"fmt"

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
	return service.ConcatStrings(
		"<telegram_message>",
		"chat_id:"+string(msg.From.ID),
		"time:"+msg.Time().Format("15:04 02.01.06"),
		"name:"+msg.From.FirstName,
		"text:"+msg.Text,
		"</telegram_message>",
	)
}
