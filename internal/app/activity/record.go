package activity

import (
	"fmt"
	"time"
)

type Record struct {
	Header  string
	Content string
}

func NewRecord(content string) Record {
	return Record{
		Header:  NowTime(),
		Content: content,
	}
}

func (r Record) Marshal() string {
	return fmt.Sprintf("## %s\n%s\n\n", r.Header, r.Content)
}

func NowTime() string {
	return time.Now().Format(TimeFormat)
}
