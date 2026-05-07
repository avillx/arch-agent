package service

import (
	"arch-agent/internal/domain/activity"
	"arch-agent/internal/domain/agent"
	"time"
)

type ActivityRepo interface {
	Log(agent.ID, activity.Record) error
	GetActivity(agent.ID, time.Time) (string, error)
}
