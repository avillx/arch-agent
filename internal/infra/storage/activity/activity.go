package activityfiles

import (
	"arch-agent/internal/app/activity"
	"arch-agent/internal/infra/storage/filesystem"
	"fmt"
	"os"
	"sync"
	"time"
)

const DateFormat = "2006-01-02"

type Storage struct {
	fs filesystem.FileSystem
	mu sync.RWMutex
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

func (f *Storage) Log(r activity.Record) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	actualFile := toFilename(time.Now())
	data := r.Marshal()
	return f.fs.AppendToFile(actualFile, []byte(data))
}

func (f *Storage) GetActivity(date time.Time) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	filename := toFilename(date)
	data, err := f.fs.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return "", activity.ErrNoActivity
		}
		return "", err
	}
	return string(data), nil
}

func toFilename(date time.Time) string {
	stringDate := date.Format(DateFormat)
	return fmt.Sprintf("%s.md", stringDate)
}
