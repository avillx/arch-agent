package dreamingadapter

import (
	"arch-agent/internal/app/dreaming"
	"arch-agent/internal/infra/storage/filesystem"
	"fmt"
	"os"
	"strings"
	"time"
)

const lastDreamedFile = "dreamed.lock"
const reportPrefix = "report_"

type DreamingFiles struct {
	fs filesystem.FileSystem
}

func NewDreamingFiles(dir string) (*DreamingFiles, error) {
	fs, err := filesystem.New(dir)
	if err != nil {
		return nil, err
	}
	return &DreamingFiles{
		fs: fs,
	}, nil
}

func (f *DreamingFiles) UpdateLastDreaming() error {
	date := time.Now().Format(time.RFC3339)
	return f.fs.WriteToFile(lastDreamedFile, []byte(date))
}

func (f *DreamingFiles) LastDreaming() (time.Time, error) {
	date, err := f.fs.ReadFile(lastDreamedFile)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, dreaming.ErrNeverDream
		}
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, string(date))
}

func (f *DreamingFiles) SaveReport(content string) error {
	filename, err := f.reportFilename()
	if err != nil {
		return err
	}

	return f.fs.WriteToFile(filename, []byte(content))
}

func (f *DreamingFiles) reportFilename() (string, error) {
	files, err := f.fs.ReadDir()
	if err != nil {
		return "", err
	}
	todayPrefix := reportPrefix + nowDate()
	count := 0
	for _, filename := range files {
		if strings.HasPrefix(filename, todayPrefix) {
			count++
		}
	}
	return fmt.Sprintf("%s_%d.md", todayPrefix, count), nil
}

func nowDate() string {
	return time.Now().Format("2006-01-02")
}
