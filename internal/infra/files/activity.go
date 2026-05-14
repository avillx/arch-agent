package files

import (
	"arch-agent/internal/domain/activity"
	"arch-agent/internal/domain/agent"
	"fmt"
	"os"
	"time"
)

type ActivityFiles struct {
	fs *FileSystem
}

func NewActivityFiles(fs *FileSystem) *ActivityFiles {
	return &ActivityFiles{fs: fs}
}

func (f *ActivityFiles) Log(id agent.ID, r activity.Record) error {
	actualFile := toActivityFilename(time.Now())
	data := []byte(r.String())
	return f.fs.AppendToFile("/"+string(id)+"/activity/"+actualFile, data)
}

func (f *ActivityFiles) GetActivity(id agent.ID, date time.Time) (string, error) {
	filename := toActivityFilename(date)
	data, err := f.fs.ReadFile("/" + string(id) + "/activity/" + filename)
	if err != nil {
		if os.IsNotExist(err) {
			return "", activity.ErrNoActivity
		}
		return "", err
	}
	return string(data), nil
}

func toActivityFilename(date time.Time) string {
	return fmt.Sprintf("%s.md", date.Format("2006-01-02"))
}
