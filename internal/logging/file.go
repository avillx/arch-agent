package logging

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

var _ io.Writer = (*LogFile)(nil)

type LogFile struct {
	filePath string

	mu sync.Mutex
}

func NewLogFile(filePath string) *LogFile {
	return &LogFile{
		filePath: filePath,
	}
}

func (w *LogFile) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	f, err := os.OpenFile(w.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return 0, fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	return f.Write(p)
}

func (s *LogFile) Trim(maxLines int) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_RDWR, 0640)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	linesCount := bytes.Count(data, []byte{'\n'})

	// no need to truncate
	if linesCount < maxLines {
		return nil
	}

	truncation := linesCount - maxLines
	idx := 0
	lines := bytes.SplitN(data, []byte{'\n'}, truncation+1)
	for _, line := range lines[:truncation] {
		// +1 for cutted "\n"
		idx += len(line) + 1
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if _, err := f.Write(data[idx:]); err != nil {
		return err
	}

	return f.Truncate(int64(len(data) - idx))
}
