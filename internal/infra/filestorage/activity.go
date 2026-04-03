package filestorage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type MDDailyActivityLogger struct {
	directory string
}

func NewMDDailyActivityLogger(dir string) *MDDailyActivityLogger {
	return &MDDailyActivityLogger{directory: dir}
}

func (l *MDDailyActivityLogger) Log(summary string) error {
	path := filepath.Join(l.directory, todayActivityFile())
	touchPath(path)

	timestamp := time.Now().Format("15:04")
	entry := fmt.Sprintf("# %s\n%s\n\n", timestamp, summary)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open activity log: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(entry)
	return err
}

type DailyActivityProvider struct {
	directory string
}

func NewDailyActivityProvider(l *MDDailyActivityLogger) *DailyActivityProvider {
	return &DailyActivityProvider{
		directory: l.directory,
	}
}

func (p *DailyActivityProvider) Today() (string, error) {
	return p.loadfile(todayActivityFile())
}
func (p *DailyActivityProvider) Yesterday() (string, error) {
	return p.loadfile(yesterdayActivityFile())
}

func (p *DailyActivityProvider) loadfile(filename string) (string, error) {
	path := filepath.Join(p.directory, filename)
	entry, err := os.ReadFile(path)

	if err == nil {
		return string(entry), nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	return "", err
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
