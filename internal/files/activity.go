package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
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
	return f.fs.AppendToFile(resolveActivityFilePath(id, time.Now()), data)
}

func (f *ActivityFiles) GetActivity(id agent.ID, date time.Time) (string, error) {
	data, err := f.fs.ReadFile(resolveActivityFilePath(id, date))
	if err != nil {
		if os.IsNotExist(err) {
			return "", types.ErrIsNotExist
		}
		return "", err
	}
	return string(data), nil
}

func resolveActivityFilePath(agentID agent.ID, t time.Time) string {
	return filepath.Join(
		fmt.Sprintf("/%s/activity", agentID),
		t.Format("2006/01/02/"),
		fmt.Sprintf("%s.md", t.Format("2006-01-02")),
	)
}
