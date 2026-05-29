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
	return f.fs.AppendToFile(resolveFilePath(id), data)
}

func (f *ActivityFiles) GetActivity(id agent.ID, date time.Time) (string, error) {
	filename := toActivityFilename(date)
	data, err := f.fs.ReadFile("/agent." + string(id) + "/activity/" + filename)
	if err != nil {
		if os.IsNotExist(err) {
			return "", agent.ErrNoActivity
		}
		return "", err
	}
	return string(data), nil
}

func toActivityFilename(date time.Time) string {
	return fmt.Sprintf("%s.md", date.Format("2006-01-02"))
}

func resolveFilePath(agentID agent.ID) string {
	now := time.Now()
	return filepath.Join(
		fmt.Sprintf("/files/activity/%s", agentID),
		now.Format("2006/01/02/"),
		toActivityFilename(now),
	)
}
