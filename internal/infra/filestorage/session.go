package filestorage

import (
	"arch-agent/internal/app/session"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileSessionRepository struct {
	dirBase
}

func NewFileSessionRepository(dir string) *FileSessionRepository {
	return &FileSessionRepository{
		dirBase: dirBase{
			directory: dir,
		},
	}
}

func (r *FileSessionRepository) Load() (*session.Session, error) {
	filepath, err := r.findActiveSessionPath()
	if os.IsNotExist(err) {
		filepath, err = r.createNewSession()
	}
	if err != nil {
		return nil, err
	}

	dto, err := r.loadDTO(filepath)
	if err != nil {
		return nil, err
	}

	return dtoToSession(dto)
}
func (r *FileSessionRepository) Save(s *session.Session) error {
	dto := sessionToDTO(s)

	filePath, err := r.findActiveSessionPath()
	if err != nil {
		return err
	}

	return r.saveDTO(dto, filePath)
}

func (r *FileSessionRepository) Drop() error {
	filePath, err := r.findActiveSessionPath()
	if err != nil {
		return err
	}
	return os.Remove(filePath)
}

func (r *FileSessionRepository) saveDTO(s SessionDTO, filePath string) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return r.saveToFile(data, filePath)
}

func (r *FileSessionRepository) createNewSession() (string, error) {
	r.touchDir(r.directory)

	id := generateSessionID()
	filePath := r.createSessionFilePath(id)

	if err := r.saveDTO(
		SessionDTO{
			ID: id,
		},
		filePath,
	); err != nil {
		return "", err
	}

	return filePath, nil
}

func (r *FileSessionRepository) loadDTO(filePath string) (SessionDTO, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return SessionDTO{}, err
	}
	var dto SessionDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return SessionDTO{}, err
	}
	return dto, nil
}

func (r *FileSessionRepository) createSessionFilePath(id string) string {
	return filepath.Join(r.directory, fmt.Sprintf("active_session_%s.json", id))
}

func (r *FileSessionRepository) findActiveSessionPath() (string, error) {
	r.touchDir(r.directory)

	entries, err := os.ReadDir(r.directory)
	if err != nil {
		return "", err
	}

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "active_session_") {
			return filepath.Join(r.directory, e.Name()), nil
		}
	}
	return "", os.ErrNotExist
}

func generateSessionID() string {
	b := make([]byte, 2)
	rand.Read(b)
	return fmt.Sprintf("%x%x", time.Now().Unix(), b)
}
