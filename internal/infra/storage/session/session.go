package sessionfiles

import (
	"arch-agent/internal/app/session"
	"arch-agent/internal/infra/storage/filesystem"
	"os"
	"sync"
)

// Save(Session)
// Load() Session
// Drop(Session)

const ActiveSessionName = "active_session.jsonl"

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

func (r *Storage) Load() (*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := r.fs.ReadFile(ActiveSessionName)
	if err != nil {
		if os.IsNotExist(err) {
			return session.NewSession(), nil
		}
		return nil, err
	}

	return unmarshalSession(data)
}
func (r *Storage) Save(s *session.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := marshalSession(s)
	if err != nil {
		return err
	}

	return r.fs.WriteToFile(ActiveSessionName, data)
}

func (r *Storage) Drop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.fs.DeleteFile(ActiveSessionName)
}
