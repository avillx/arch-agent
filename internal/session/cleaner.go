package session

import (
	"arch-agent/internal/agent"
	"errors"
	"log/slog"
	"sync"
	"time"
)

type Cleaner struct {
	agentRepo       agent.Repo
	sessRepo        SessionsRepo
	sessStorePeriod int
	logger          *slog.Logger

	mu sync.Mutex
}

func NewCleaner(
	agentRepo agent.Repo,
	sessRepo SessionsRepo,
	sessStorePeriod int,
	logger *slog.Logger,
) *Cleaner {
	return &Cleaner{
		agentRepo:       agentRepo,
		sessRepo:        sessRepo,
		sessStorePeriod: sessStorePeriod,
		logger:          logger.WithGroup("session_cleaner"),
	}
}

func (c *Cleaner) Clean() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	agents, err := c.agentRepo.All()
	if err != nil {
		return err
	}

	for _, agt := range agents {
		headers, err := c.sessRepo.Headers(agt.ID())
		if err != nil {

			var brokenHeadersErr *ErrBrokenHeaders
			if !errors.As(err, &brokenHeadersErr) {
				c.logger.Error("read session headers", "agent", agt.ID(), "error", err)
				continue
			}

			c.handleBrokenSessions(brokenHeadersErr)

		}

		c.eliminateDepricatedSessions(agt.ID(), headers)
	}

	return nil
}

func (c *Cleaner) handleBrokenSessions(brokenHeadersErr *ErrBrokenHeaders) {
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

func (c *Cleaner) eliminateDepricatedSessions(agentID agent.ID, headers []SessionHeader) {
	cutDate := time.Now().AddDate(0, 0, -c.sessStorePeriod)
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
