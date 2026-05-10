package files

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/session"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type SessionFiles struct {
	fs           *FileSystem
	tokenCounter session.TokenCounter
}

func NewSessionFiles(fs *FileSystem) *SessionFiles {
	return &SessionFiles{fs: fs}
}

func (r *SessionFiles) Session(agentID agent.ID, sesstionID session.ID) (*session.Session, error) {

	sessionFilePath := fmt.Sprintf("/%s/sessions/%s.json", agentID, sesstionID)

	data, err := r.fs.ReadFile(sessionFilePath)
	if err != nil {
		return nil, err
	}

	return r.unmarshalSession(sesstionID, data)
}

func (r *SessionFiles) Save(agentID agent.ID, s *session.Session) error {
	data, err := marshalSession(s)
	if err != nil {
		return err
	}

	sessionFilePath := fmt.Sprintf("/%s/sessions/%s.json", agentID, string(s.ID))
	return r.fs.WriteToFile(sessionFilePath, data)
}

func (r *SessionFiles) Delete(agentID agent.ID, id session.ID) error {
	sessionFilePath := fmt.Sprintf("/%s/sessions/%s.json", agentID, string(id))
	return r.fs.DeleteFile(sessionFilePath)
}

func (r *SessionFiles) List(agentID agent.ID) ([]session.ID, error) {

	sessionDir := fmt.Sprintf("/%s/sessions", agentID)

	filenames, err := r.fs.ReadDir(sessionDir)
	if err != nil {
		return nil, err
	}

	sessionIDs := make([]session.ID, len(filenames))
	for _, filename := range filenames {
		sessionID := strings.TrimRight(filename, filepath.Ext(filename))
		sessionIDs = append(sessionIDs, session.ID(sessionID))
	}

	return sessionIDs, nil
}

func (r *SessionFiles) unmarshalSession(id session.ID, data []byte) (*session.Session, error) {
	var dto SessionDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return r.dtoToSession(id, dto)
}

func (r *SessionFiles) dtoToSession(id session.ID, dto SessionDTO) (*session.Session, error) {
	msgs, err := DtoToMessages(dto.Messages)
	if err != nil {
		return nil, err
	}
	return session.NewRestoredSession(id, dto.Tokens, msgs, r.tokenCounter, nil), nil
}

type SessionDTO struct {
	Tokens   int          `json:"tokens"`
	Messages []MessageDTO `json:"messages"`
}

func marshalSession(s *session.Session) ([]byte, error) {
	return json.MarshalIndent(SessionDTO{
		Tokens:   s.Tokens,
		Messages: MessagesToDTO(s.Messages()),
	}, "", "	")
}
