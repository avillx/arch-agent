package filestorage

import (
	"arch-agent/internal/app/message"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type JSONLTranscriber struct {
	dirBase
}

func NewJSONLTranscriber(dir string) *JSONLTranscriber {
	return &JSONLTranscriber{
		dirBase: dirBase{
			directory: dir,
		},
	}
}

func (t *JSONLTranscriber) Transcribe(sessionID string, messages []message.Message) error {
	path := t.filePath(sessionID)

	t.touchDir(path)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open transcription file: %w", err)
	}
	defer f.Close()

	return t.writeMessages(json.NewEncoder(f), messages)
}

func (t *JSONLTranscriber) writeMessages(enc *json.Encoder, messages []message.Message) error {
	for _, dto := range messagesToDTO(messages) {
		if err := enc.Encode(dto); err != nil {
			return fmt.Errorf("encode message: %w", err)
		}
	}
	return nil
}

func (t *JSONLTranscriber) filePath(sessionID string) string {
	return filepath.Join(t.directory, transcriptionFileName(sessionID))
}

func transcriptionFileName(sessionID string) string {
	date := time.Now().Format("02.01.06")
	return fmt.Sprintf("%s_%s.jsonl", date, sessionID)
}
