package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var _ agent.ActivityRepo = (*ActivityFiles)(nil)

type ActivityFiles struct {
	fs *FileSystem
}

func NewActivityFiles(fs *FileSystem) *ActivityFiles {
	return &ActivityFiles{fs: fs}
}

func (f *ActivityFiles) Log(id agent.ID, r agent.ActivityRecord) error {
	data := []byte(r.String())
	return f.fs.AppendToFile(resolveActivityFilePath(id, time.Now()), data)
}

func (f *ActivityFiles) GetActivity(id agent.ID, date time.Time) (string, error) {
	data, err := f.fs.ReadFile(resolveActivityFilePath(id, date))
	if err != nil {
		if os.IsNotExist(err) {
			return "", types.ErrIsNotExist
		}
		return "", err
	}
	return string(data), nil
}

func (f *ActivityFiles) GetRange(
	agentID agent.ID,
	from time.Time,
	to time.Time,
) ([]agent.ActivityLog, error) {
	logs := []agent.ActivityLog{}

	if to.IsZero() {
		to = time.Now()
	}

	for i := from; !i.After(to); i = i.AddDate(0, 0, 1) {

		p := resolveActivityFilePath(agentID, i)
		data, err := f.fs.ReadFile(p)
		if err != nil && !errors.Is(err, types.ErrIsNotExist) {
			return nil, err
		}
		if data != nil {
			logs = append(logs, agent.ActivityLog{
				Date:    i,
				Content: string(data),
			})
		}
	}

	return logs, nil
}

func resolveActivityFilePath(agentID agent.ID, t time.Time) string {
	d := t.UTC().Truncate(24 * time.Hour)
	return filepath.Join(
		fmt.Sprintf("/%s/activity", agentID),
		d.Format("2006"),
		d.Format("01"),
		d.Format("02"),
		fmt.Sprintf("%s.md", d.Format("2006-01-02")),
	)
}
