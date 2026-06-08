package files

import (
	"arch-agent/internal/agent"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var _ agent.ActivityRepo = (*ActivityFiles)(nil)

type ActivityFiles struct {
	fs *FileSystem
}

func NewActivityFiles(fs *FileSystem) *ActivityFiles {
	return &ActivityFiles{fs: fs}
}

func (f *ActivityFiles) Log(id agent.ID, r agent.ActivityRecord) error {
	data := []byte(r.String())
	return f.fs.AppendToFile(resolveFilePath(id, time.Now()), data)
}

func (f *ActivityFiles) GetActivity(id agent.ID, date time.Time) (string, error) {
	data, err := f.fs.ReadFile(resolveFilePath(id, date))
	if err != nil {
		if os.IsNotExist(err) {
			return "", agent.ErrNoActivity
		}
		return "", err
	}
	return string(data), nil
}

func resolveFilePath(agentID agent.ID, t time.Time) string {
	return filepath.Join(
		fmt.Sprintf("/files/activity/%s", agentID),
		t.Format("2006/01/02/"),
		fmt.Sprintf("%s.md", t.Format("2006-01-02")),
	)
}
