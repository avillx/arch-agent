package files

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/session"
	"encoding/json"
	"os"
)

type SessionFiles struct {
	fs *FileSystem
}

func NewSessionFiles(fs *FileSystem) *SessionFiles {
	return &SessionFiles{fs: fs}
}

func (r *SessionFiles) Session(id session.ID) (session.Session, error) {
	data, err := r.fs.ReadFile(string(id) + ".json")
	if err != nil {
		return session.Session{}, err
	}

	return unmarshalSession(data)
}

func (r *SessionFiles) Save(s session.Session) error {
	data, err := marshalSession(s)
	if err != nil {
		return err
	}

	return r.fs.WriteToFile(string(s.ID)+".json", data)
}

func (r *SessionFiles) Delete(id session.ID) error {
	return r.fs.DeleteFile(string(id) + ".json")
}

func (r *SessionFiles) List(_ agent.ID) ([]session.Session, error) {
	names, err := r.fs.ReadDir()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	sessions := make([]session.Session, 0, len(names))
	for _, n := range names {
		if len(n) <= 5 || n[len(n)-5:] != ".json" {
			continue
		}
		data, err := r.fs.ReadFile(n)
		if err != nil {
			continue
		}
		s, err := unmarshalSession(data)
		if err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

type SessionDTO struct {
	ID       string       `json:"id"`
	Tokens   int          `json:"tokens"`
	Messages []MessageDTO `json:"messages"`
}

func unmarshalSession(data []byte) (session.Session, error) {
	var dto SessionDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return session.Session{}, err
	}
	return dtoToSession(dto)
}

func dtoToSession(dto SessionDTO) (session.Session, error) {
	msgs, err := DtoToMessages(dto.Messages)
	if err != nil {
		return session.Session{}, err
	}
	return *session.NewRestoredSession(dto.Tokens, msgs, nil), nil
}

func marshalSession(s session.Session) ([]byte, error) {
	return json.Marshal(SessionDTO{
		ID:       string(s.ID),
		Tokens:   s.Tokens,
		Messages: MessagesToDTO(s.Messages()),
	})
}
