package transcribtions

import (
	"arch-agent/internal/app/types"
	"arch-agent/internal/infra/storage"
	"arch-agent/internal/infra/storage/filesystem"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"
)

type JSONLTranscriber struct {
	fs filesystem.FileSystem
}

func NewJSONLTranscriber(dir string) (*JSONLTranscriber, error) {
	fs, err := filesystem.New(dir)
	if err != nil {
		return nil, err
	}
	return &JSONLTranscriber{
		fs: fs,
	}, nil
}

func (t *JSONLTranscriber) Transcribe(messages []types.Message) error {
	data, err := json.Marshal(storage.MessagesToDTO(messages))
	if err != nil {
		return err
	}
	return t.fs.WriteToFile(transcriptionFileName(), data)
}

func transcriptionFileName() string {
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s_%s.jsonl", date, generateUUID())
}

func generateUUID() string {
	b := make([]byte, 2)
	rand.Read(b)
	return fmt.Sprintf("%x%x", time.Now().Unix(), b)
}
