package activityadapter

import (
	"arch-agent/internal/app/activity"
	"arch-agent/internal/infra/storage/filesystem"
	"fmt"
	"os"
	"time"
)

const DateFormat = "2006-01-02"

type ActivityFiles struct {
	fs filesystem.FileSystem
}

func NewActivityFiles(dir string) (*ActivityFiles, error) {
	fs, err := filesystem.New(dir)
	if err != nil {
		return nil, err
	}
	return &ActivityFiles{
		fs: fs,
	}, nil
}

func (f *ActivityFiles) Log(r activity.Record) error {
	actualFile := toFilename(time.Now())
	data := r.Marshal()
	return f.fs.AppendToFile(actualFile, []byte(data))
}

func (f *ActivityFiles) GetActivity(date time.Time) (string, error) {
	filename := toFilename(date)
	data, err := f.fs.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return "", activity.ErrNotFound
		}
		return "", err
	}
	return string(data), nil
}

func toFilename(date time.Time) string {
	stringDate := date.Format(DateFormat)
	return fmt.Sprintf("%s.md", stringDate)
}
