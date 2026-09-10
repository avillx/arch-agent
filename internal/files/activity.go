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

const ActivityFolder = "activity"

type ActivityFiles struct {
	storage FileStorage
}

func NewActivityFiles(storage FileStorage) *ActivityFiles {
	return &ActivityFiles{
		storage: storage,
	}
}

func (a *ActivityFiles) Log(id agent.ID, r agent.ActivityRecord) error {
	data := []byte(r.String())

	p := resolveActivityFilePath(id, time.Now())
	f, err := a.storage.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, ModeAppend)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

func (a *ActivityFiles) GetActivity(id agent.ID, date time.Time) (string, error) {
	data, err := a.storage.ReadFile(resolveActivityFilePath(id, date))
	if err != nil {
		if os.IsNotExist(err) {
			return "", types.ErrIsNotExist
		}
		return "", err
	}
	return string(data), nil
}

func (a *ActivityFiles) GetRange(
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
		data, err := a.storage.ReadFile(p)
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
		string(agentID),
		ActivityFolder,
		d.Format("2006"),
		d.Format("01"),
		fmt.Sprintf("%s.md", d.Format("02")),
	)
}
