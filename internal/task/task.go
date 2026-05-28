package task

import (
	"arch-agent/internal/agent"
	"time"
)

type Cron interface {
	NextTime() time.Duration
	Expression() string
}

type Task struct {
	Name        string
	Recipients  []agent.ID
	Description string
	Request     string
	Reglament   Cron
	OneShot     bool
}

func NewTask(
	name string,
	description string,
	recipients []agent.ID,
	request string,
	reglament Cron,
	oneShot bool,
) Task {
	return Task{
		Name:        name,
		Description: description,
		Recipients:  recipients,
		Request:     request,
		Reglament:   reglament,
		OneShot:     oneShot,
	}
}
