package cleanup

import (
	"arch-agent/internal/logging"
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	minSessRetention   = 10 * time.Hour
	minCleanUpInterval = 5 * time.Hour
)

type CleanUpConfig struct {
	SessRetention   time.Duration
	MaxLogLines     int
	CleanUpInterval time.Duration
}

type CleanUpService struct {
	sessionsCleaner *SessionsCleaner
	logFile         *logging.LogFile
	logger          *slog.Logger

	cfg CleanUpConfig
}

func NewCleanUpService(
	cfg CleanUpConfig,
	sessionsCleaner *SessionsCleaner,
	logFile *logging.LogFile,
	logger *slog.Logger,
) (*CleanUpService, error) {

	if cfg.SessRetention < minSessRetention {
		return nil, fmt.Errorf("too short session retention, must be greather than 24h")
	}

	if cfg.CleanUpInterval < minCleanUpInterval {
		return nil, fmt.Errorf("too short cleanup interval, must be greather than 5h")
	}

	return &CleanUpService{
		cfg:             cfg,
		sessionsCleaner: sessionsCleaner,
		logFile:         logFile,
		logger:          logger.WithGroup("cleanup"),
	}, nil
}

func (s *CleanUpService) Run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.CleanUpInterval)
	defer ticker.Stop()

	// clean immidiate
	s.doCleanUp()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.doCleanUp()
		}
	}
}

func (s *CleanUpService) doCleanUp() {
	s.logger.Error("invoke")

	s.logger.Info("cleaning sessions")
	if err := s.sessionsCleaner.Clean(s.cfg.SessRetention); err != nil {
		s.logger.Error("sessions", "error", err)
	}

	s.logger.Info("cleaning log")
	if err := s.logFile.Trim(s.cfg.MaxLogLines); err != nil {
		s.logger.Error("agent log file", "error", err)
	}
}
