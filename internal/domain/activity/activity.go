package activity

import (
	"errors"
	"fmt"
	"time"
)

const TimeFormat = "15:04"

var ErrNoActivity = errors.New("Activity on date is not found")

type Record struct {
	Stamp   time.Time
	Content string
}

func NewRecord(content string) Record {
	return Record{
		Stamp:   time.Now(),
		Content: content,
	}
}

func (r Record) String() string {
	timeHeader := time.Now().Format(TimeFormat)
	return fmt.Sprintf("## %s\n%s\n\n", timeHeader, r.Content)
}
