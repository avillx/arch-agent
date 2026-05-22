package agent

import (
	"errors"
	"fmt"
	"time"
)

type ActivityRepo interface {
	Log(ID, ActivityRecord) error
	GetActivity(ID, time.Time) (string, error)
}

const TimeFormat = "15:04"

var ErrNoActivity = errors.New("Activity on date is not found")

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
