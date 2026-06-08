package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var _ session.SessionsRepo = (*SessionFiles)(nil)

type SessionFiles struct {
	fs *FileSystem
}

func NewSessionFiles(fs *FileSystem) *SessionFiles {
	return &SessionFiles{
		fs: fs,
	}
}

func (r *SessionFiles) Session(agentID agent.ID, sesstionID session.ID) (session.Session, error) {

	sessionFilePath := fmt.Sprintf("/agent.%s/sessions/%s.json", agentID, sesstionID)

	data, err := r.fs.ReadFile(sessionFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, types.ErrIsNotExist
		}
		return nil, err
	}

	return r.unmarshalSession(sesstionID, data)
}

func (r *SessionFiles) Save(agentID agent.ID, s session.Session) error {
	data, err := marshalSession(s)
	if err != nil {
		return err
	}

	sessionFilePath := fmt.Sprintf("/agent.%s/sessions/%s.json", agentID, string(s.ID()))
	return r.fs.WriteToFile(sessionFilePath, data)
}

func (r *SessionFiles) Delete(agentID agent.ID, id session.ID) error {
	sessionFilePath := fmt.Sprintf("/agent.%s/sessions/%s.json", agentID, string(id))
	return r.fs.Delete(sessionFilePath)
}

func (r *SessionFiles) List(agentID agent.ID) ([]session.ID, error) {

	sessionDir := fmt.Sprintf("/agent.%s/sessions", agentID)

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

func (r *SessionFiles) unmarshalSession(id session.ID, data []byte) (session.Session, error) {
	var dto SessionDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return r.dtoToSession(id, dto)
}

func (r *SessionFiles) dtoToSession(id session.ID, dto SessionDTO) (session.Session, error) {
	msgs, err := DtoToMessages(dto.Messages)
	if err != nil {
		return nil, err
	}
	return session.NewRestoredSession(
		id,
		msgs,
		dto.InputTokens,
		dto.OutputTokens,
		dto.Summary,
		dto.SubSessions,
		dto.CreatedAt,
	), nil
}

type SessionDTO struct {
	InputTokens  int64                   `json:"input_tokens"`
	OutputTokens int64                   `json:"output_tokens"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	Summary      string                  `json:"summary"`
	Messages     []MessageDTO            `json:"messages"`
	SubSessions  map[agent.ID]session.ID `json:"sub_sessions"`
}

func marshalSession(s session.Session) ([]byte, error) {
	return json.MarshalIndent(SessionDTO{
		InputTokens:  s.InputTokens(),
		OutputTokens: s.OutputTokens(),
		CreatedAt:    s.CreatedAt(),
		UpdatedAt:    s.UpdatedAt(),
		Summary:      s.Summary(),
		Messages:     MessagesToDTO(s.Messages()),
		SubSessions:  s.Subsessions(),
	}, "", "	")
}
