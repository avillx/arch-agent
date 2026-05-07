package service

import (
	"arch-agent/internal/domain/activity"
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/session"
	"time"
)

type ActivityRepo interface {
	Log(activity.Record) error
	GetActivity(date time.Time) (string, error)
}

type SessionsRepo interface {
	List(agent.ID) []session.Session
	Session(SessionID session.ID) session.Session
	Save(SessionVO session.Session)
	Delete(SessionID session.ID)
}
