package agent

import (
	"fmt"
	"time"
)

const TimeFormat = "15:04"

type ActivityLog struct {
	// at first is a date. current time is not provided
	Date    time.Time
	Content string
}

type ActivityRepo interface {
	Log(ID, ActivityRecord) error
	GetActivity(ID, time.Time) (string, error)
	GetRange(ID, time.Time, time.Time) ([]ActivityLog, error)
}

type MemoryRepo interface {
	GetMemory(ID, string) (string, error)
}

type MemoryIndexer interface {
	MemoryIndex(ID) (map[string]string, error)
}

type ActivityRecord struct {
	Stamp   time.Time
	Content string
}

func NewRecord(content string) ActivityRecord {
	return ActivityRecord{
		Stamp:   time.Now(),
		Content: content,
	}
}

func (r ActivityRecord) String() string {
	timeHeader := time.Now().Format(TimeFormat)
	return fmt.Sprintf("## %s\n%s\n\n", timeHeader, r.Content)
}
