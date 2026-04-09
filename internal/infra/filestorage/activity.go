package filestorage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type MDDailyActivityStore struct {
	dirBase
}

func NewMDDailyActivityStore(dir string) *MDDailyActivityStore {
	return &MDDailyActivityStore{
		dirBase: dirBase{
			directory: dir,
		},
	}
}

func (s *MDDailyActivityStore) Log(summary string) error {
	s.touchDir(s.directory)
	path := s.todayActivityFilePath()
	record := createRecord(summary)

	return appendToFile(path, record)
}

func (s *MDDailyActivityStore) Today() (string, error) {
	return s.loadfile(todayActivityFile())
}
func (s *MDDailyActivityStore) Yesterday() (string, error) {
	return s.loadfile(yesterdayActivityFile())
}

func (s *MDDailyActivityStore) todayActivityFilePath() string {
	return filepath.Join(s.directory, todayActivityFile())
}

func (s *MDDailyActivityStore) loadfile(filename string) (string, error) {
	path := filepath.Join(s.directory, filename)
	entry, err := os.ReadFile(path)

	if err == nil {
		return string(entry), nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	return "", err
}

func createRecord(summary string) string {
	timestamp := time.Now().Format("15:04")
	return fmt.Sprintf("# %s\n%s\n\n", timestamp, summary)
}

// helpers
func todayActivityFile() string {
	date := time.Now().Format("02.01.06")
	return fmt.Sprintf("%s.md", date)
}

func yesterdayActivityFile() string {
	date := time.Now().AddDate(0, 0, -1).Format("02.01.06")
	return fmt.Sprintf("%s.md", date)
}

func appendToFile(path, record string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open activity log: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(record)
	return err
}
