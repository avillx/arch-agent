package cleanup

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"errors"
	"log/slog"
	"sync"
	"time"
)

type SessionsCleaner struct {
	agentRepo agent.Repo
	sessRepo  session.SessionsRepo
	logger    *slog.Logger

	mu sync.Mutex
}

func NewSessionsCleaner(
	agentRepo agent.Repo,
	sessRepo session.SessionsRepo,

	logger *slog.Logger,
) *SessionsCleaner {
	return &SessionsCleaner{
		agentRepo: agentRepo,
		sessRepo:  sessRepo,
		logger:    logger.WithGroup("session_cleaner"),
	}
}

func (c *SessionsCleaner) Clean(retention time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	agents, err := c.agentRepo.All()
	if err != nil {
		return err
	}

	for _, agt := range agents {
		headers, err := c.sessRepo.Headers(agt.ID())
		if err != nil {

			var brokenHeadersErr *session.ErrBrokenHeaders
			if !errors.As(err, &brokenHeadersErr) {
				c.logger.Error("read session headers", "agent", agt.ID(), "error", err)
				continue
			}

			c.handleBrokenSessions(brokenHeadersErr)

		}

		c.eliminateDepricatedSessions(retention, agt.ID(), headers)
	}

	return nil
}

func (c *SessionsCleaner) handleBrokenSessions(brokenHeadersErr *session.ErrBrokenHeaders) {
	for _, brokenHeaderErr := range brokenHeadersErr.Errors {
		logger := c.logger.With(
			"agent", brokenHeaderErr.AgentID,
			"session", brokenHeaderErr.SessID,
			"cause", brokenHeaderErr.Unwrap(),
		)

		if err := c.sessRepo.Delete(brokenHeaderErr.AgentID, brokenHeaderErr.SessID); err != nil {
			logger.Error("deletion broken session", "error", err)
			continue
		}

		logger.Warn("broken session deleted")
	}
}

func (c *SessionsCleaner) eliminateDepricatedSessions(
	retention time.Duration,
	agentID agent.ID,
	headers []session.SessionHeader,
) {
	cutDate := time.Now().Add(-retention)
	for _, h := range headers {
		if h.UpdatedAt().After(cutDate) {
			continue
		}

		logger := c.logger.With("agent", agentID, "session", h.ID())

		if err := c.sessRepo.Delete(agentID, h.ID()); err != nil {
			logger.Error("deletion deprecated session", "error", err)
			continue
		}
		logger.Info("deprecated session deleted")
	}
}
