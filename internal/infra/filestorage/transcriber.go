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
	directory string
}

func NewJSONLTranscriber(dir string) *JSONLTranscriber {
	return &JSONLTranscriber{directory: dir}
}

func (t *JSONLTranscriber) Transcribe(sessionID string, messages []message.Message) error {
	path := t.filePath(sessionID)

	touchPath(path)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open transcription file: %w", err)
	}
	defer f.Close()

	return t.writeMessages(json.NewEncoder(f), messages)
}

func (t *JSONLTranscriber) writeMessages(enc *json.Encoder, messages []message.Message) error {
	for _, m := range messages {
		record, err := messageToDTO(m)
		if err != nil {
			return err
		}
		if err := enc.Encode(record); err != nil {
			return fmt.Errorf("encode message: %w", err)
		}
	}
	return nil
}

func (t *JSONLTranscriber) filePath(sessionID string) string {
	return filepath.Join(t.directory, sessionFileName(sessionID))
}

func sessionFileName(sessionID string) string {
	date := time.Now().Format("02.01.06")
	return fmt.Sprintf("%s_%s.jsonl", date, sessionID)
}
