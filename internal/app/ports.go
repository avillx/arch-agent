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
	List(agent.ID) ([]*session.Session, error)
	Session(SessionID session.ID) (*session.Session, error)
	Save(Session *session.Session) error
	Delete(SessionID session.ID) error
}
