package transcribtionfiles

import (
	"arch-agent/internal/app/types"
	"arch-agent/internal/infra/storage"
	"arch-agent/internal/infra/storage/filesystem"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Storage struct {
	fs filesystem.FileSystem
	mu sync.Mutex
}

func New(dir string) (*Storage, error) {
	fs, err := filesystem.New(dir)
	if err != nil {
		return nil, err
	}
	return &Storage{
		fs: fs,
	}, nil
}

func (t *Storage) Transcribe(messages []types.Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()

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
