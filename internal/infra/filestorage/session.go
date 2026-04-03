package filestorage

import (
	"arch-agent/internal/app/message"
	"arch-agent/internal/app/session"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileSessionRepository struct {
	directory string
}

func NewFileSessionRepository(dir string) *FileSessionRepository {
	return &FileSessionRepository{directory: dir}
}

func (r *FileSessionRepository) Load() *session.Session {
	entries, err := os.ReadDir(r.directory)
	if err != nil {
		slog.Error("read session dir", "err", err)
		return r.create()
	}

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "active_session_") {
			return r.loadFile(filepath.Join(r.directory, e.Name()))
		}
	}

	return r.create()
}

func (r *FileSessionRepository) Update(s *session.Session) error {
	data, err := MarshalSession(s)
	if err != nil {
		return err
	}

	path := r.filePath(s.ID())
	touchPath(path)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	return nil
}

func (r *FileSessionRepository) Drop(s *session.Session) {
	if err := os.Remove(r.filePath(s.ID())); err != nil {
		slog.Error("drop session", "err", err)
	}
}

func (r *FileSessionRepository) create() *session.Session {
	id := generateSessionID()
	return session.NewSession(id, 0, []message.Message{})
}

func (r *FileSessionRepository) loadFile(path string) *session.Session {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("read session file", "err", err)
		return r.create()
	}
	s, err := UnmarshalSession(data)
	if err != nil {
		slog.Error("unmarshal session", "err", err)
		return r.create()
	}
	return s
}

func (r *FileSessionRepository) filePath(id string) string {
	return filepath.Join(r.directory, fmt.Sprintf("active_session_%s.json", id))
}

func generateSessionID() string {
	b := make([]byte, 2)
	rand.Read(b)
	return fmt.Sprintf("%x%x", time.Now().Unix(), b)
}
