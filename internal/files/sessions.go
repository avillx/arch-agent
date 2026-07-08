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

func (r *SessionFiles) Session(agentID agent.ID, sessionID session.ID) (session.Session, error) {

	sessionFilePath := resolveSessionPath(agentID, sessionID)

	data, err := r.fs.ReadFile(sessionFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, types.ErrIsNotExist
		}
		return nil, err
	}

	return r.unmarshalSession(sessionID, data)
}

func (r *SessionFiles) Save(agentID agent.ID, s session.Session) error {
	data, err := marshalSession(s)
	if err != nil {
		return err
	}

	sessionFilePath := resolveSessionPath(agentID, s.ID())
	return r.fs.WriteToFile(sessionFilePath, data)
}

func (r *SessionFiles) Delete(agentID agent.ID, sessionID session.ID) error {
	sessionFilePath := resolveSessionPath(agentID, sessionID)
	return r.fs.Delete(sessionFilePath)
}

func (r *SessionFiles) List(agentID agent.ID) ([]session.ID, error) {

	sessionDir := resolveSessionFolderPath(agentID)
	files, err := r.fs.ReadDir(sessionDir)
	if err != nil {
		return nil, err
	}

	sessionIDs := make([]session.ID, len(files))
	for _, file := range files {

		fileName := file.Name()

		sessionID := strings.TrimRight(fileName, filepath.Ext(fileName))
		sessionIDs = append(sessionIDs, session.ID(sessionID))
	}

	return sessionIDs, nil
}

func (r *SessionFiles) unmarshalSession(sessionID session.ID, data []byte) (session.Session, error) {
	var dto SessionDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return r.dtoToSession(sessionID, dto)
}

func (r *SessionFiles) dtoToSession(sessionID session.ID, dto SessionDTO) (session.Session, error) {
	msgs, err := DtoToMessages(dto.Messages)
	if err != nil {
		return nil, err
	}
	return session.NewRestoredSession(
		sessionID,
		msgs,
		dto.InputTokens,
		dto.OutputTokens,
		dto.Summary,
		dto.CreatedAt,
		dto.Extras,
	), nil
}

type SessionDTO struct {
	InputTokens  int64          `json:"input_tokens"`
	OutputTokens int64          `json:"output_tokens"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Summary      string         `json:"summary"`
	Messages     []MessageDTO   `json:"messages"`
	Extras       map[string]any `json:"extras"`
}

func marshalSession(s session.Session) ([]byte, error) {
	return json.MarshalIndent(SessionDTO{
		InputTokens:  s.InputTokens(),
		OutputTokens: s.OutputTokens(),
		CreatedAt:    s.CreatedAt(),
		UpdatedAt:    s.UpdatedAt(),
		Summary:      s.Summary(),
		Messages:     MessagesToDTO(s.Messages()),
		Extras:       s.Extras(),
	}, "", "	")
}

func resolveSessionPath(agentID agent.ID, sessionID session.ID) string {
	return filepath.Join(resolveSessionFolderPath(agentID), fmt.Sprintf("%s.json", sessionID))
}
func resolveSessionFolderPath(agentID agent.ID) string {
	return fmt.Sprintf("/%s/sessions", agentID)
}
