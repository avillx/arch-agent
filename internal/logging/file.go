package logging

import (
	"arch-agent/internal/files"
	"bytes"
	"fmt"
	"io"
	"sync"
)

const AgentLogFile = "agent.log"

var _ io.Writer = (*LogFile)(nil)

type LogFile struct {
	storage  files.FileStorage
	fileName string

	mu sync.Mutex
}

func NewLogFile(storage files.FileStorage) *LogFile {
	return &LogFile{
		storage:  storage,
		fileName: AgentLogFile,
	}
}

func (s *LogFile) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.storage.OpenFile(s.fileName, files.O_APPEND|files.O_CREATE|files.O_WRONLY, 0640)
	if err != nil {
		return 0, fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	return f.Write(p)
}

func (s *LogFile) Trim(maxLines int) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.storage.OpenFile(s.fileName, files.O_CREATE|files.O_RDWR, 0640)
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
