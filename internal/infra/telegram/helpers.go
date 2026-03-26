package telegram

import (
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
