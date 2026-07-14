package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	return unmarshalSession(sessionID, data)
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

type SessionHeaderDTO struct {
	InputTokens  int64          `json:"input_tokens"`
	OutputTokens int64          `json:"output_tokens"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Extras       map[string]any `json:"extras"`
}

func marshalSession(s session.Session) ([]byte, error) {
	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)

	if err := enc.Encode(SessionHeaderDTO{
		InputTokens:  s.InputTokens(),
		OutputTokens: s.OutputTokens(),
		CreatedAt:    s.CreatedAt(),
		UpdatedAt:    s.UpdatedAt(),
		Extras:       s.Extras(),
	}); err != nil {
		return nil, err
	}

	for _, m := range MessagesToDTO(s.Messages()) {
		if err := enc.Encode(m); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func unmarshalSession(sessionID session.ID, data []byte) (session.Session, error) {

	dec := json.NewDecoder(bytes.NewReader(data))

	var header SessionHeaderDTO
	if err := dec.Decode(&header); err != nil {
		return nil, err
	}

	var msgs []MessageDTO
	for {
		var m MessageDTO
		err := dec.Decode(&m)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		msgs = append(msgs, m)
	}

	return dtoToSession(sessionID, header, msgs)
}

func dtoToSession(sessionID session.ID, headerDTO SessionHeaderDTO, messagesDTO []MessageDTO) (session.Session, error) {
	msgs, err := DtoToMessages(messagesDTO)
	if err != nil {
		return nil, err
	}
	return session.NewRestoredSession(
		sessionID,
		msgs,
		headerDTO.InputTokens,
		headerDTO.OutputTokens,
		headerDTO.CreatedAt,
		headerDTO.Extras,
	), nil
}

func resolveSessionPath(agentID agent.ID, sessionID session.ID) string {
	return filepath.Join(resolveSessionFolderPath(agentID), fmt.Sprintf("%s.jsonl", sessionID))
}
func resolveSessionFolderPath(agentID agent.ID) string {
	return fmt.Sprintf("/%s/sessions", agentID)
}
