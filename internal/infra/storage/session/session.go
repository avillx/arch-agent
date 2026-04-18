package sessionadapter

import (
	"arch-agent/internal/app/session"
	"arch-agent/internal/infra/storage/filesystem"
	"os"
)

// Save(Session)
// Load() Session
// Drop(Session)

const ActiveSessionName = "active_session.jsonl"

type SessionFiles struct {
	fs filesystem.FileSystem
}

func NewFileSessionRepository(dir string) *SessionFiles {
	return &SessionFiles{
		fs: filesystem.New(dir),
	}
}

func (r *SessionFiles) Load() (*session.Session, error) {
	data, err := r.fs.ReadFile(ActiveSessionName)
	if err != nil && os.IsNotExist(err) {
		return session.NewSession(), nil
	}
	if err != nil {
		return nil, err
	}

	return unmarshalSession(data)
}
func (r *SessionFiles) Save(s *session.Session) error {
	data, err := marshalSession(s)
	if err != nil {
		return err
	}

	return r.fs.WriteToFile(ActiveSessionName, data)
}

func (r *SessionFiles) Drop() error {
	return r.fs.DeleteFile(ActiveSessionName)
}
