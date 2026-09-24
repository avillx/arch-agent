package cleanup_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/cleanup"
	"arch-agent/internal/session"
	"context"
	"errors"
	"log/slog"
	"reflect"
	"testing"
	"time"
)

func runSinglePass(t *testing.T, svc *cleanup.CleanUpService, signal <-chan struct{}) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()

	select {
	case <-signal:
		cancel()
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("cleanup pass was not invoked")
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop after context cancel")
	}
}

func newService(t *testing.T, sessRepo session.SessionsRepo, logFile *mockLogTrimmer, logger *slog.Logger) *cleanup.CleanUpService {
	t.Helper()

	svc, err := cleanup.NewCleanUpService(cleanup.CleanUpConfig{
		SessRetention:   testRetention,
		CleanUpInterval: testInterval,
		MaxLogLines:     testMaxLogLines,
	}, cleanup.NewSessionsCleaner(
		newMockAgentRepo(agent.NewAgent(testAgentID, "", "", "", nil, false)),
		sessRepo,
		logger,
	), logFile, logger)
	if err != nil {
		t.Fatalf("create cleanup service: %v", err)
	}
	return svc
}

func TestNewCleanUpService_RejectsShortRetention(t *testing.T) {
	_, err := cleanup.NewCleanUpService(cleanup.CleanUpConfig{
		SessRetention:   time.Hour,
		CleanUpInterval: testInterval,
		MaxLogLines:     testMaxLogLines,
	}, nil, nil, slog.New(slog.DiscardHandler))

	if err == nil {
		t.Fatal("expected an error for too short session retention")
	}
}

func TestNewCleanUpService_RejectsShortInterval(t *testing.T) {
	_, err := cleanup.NewCleanUpService(cleanup.CleanUpConfig{
		SessRetention:   testRetention,
		CleanUpInterval: time.Hour,
		MaxLogLines:     testMaxLogLines,
	}, nil, nil, slog.New(slog.DiscardHandler))

	if err == nil {
		t.Fatal("expected an error for too short cleanup interval")
	}
}

func TestRun_CleansImmediatelyAndStopsOnCancel(t *testing.T) {
	old := time.Now().Add(-24 * time.Hour)
	sessRepo := &mockSessionsRepo{
		headersByAgent: map[agent.ID][]session.SessionHeader{
			testAgentID: {
				oldHeader("sess-old", old),
				oldHeader("sess-new", time.Now()),
			},
		},
	}
	trimmer := &mockLogTrimmer{trimmed: make(chan struct{}, 4)}
	logger, handler := newStubLogger()
	svc := newService(t, sessRepo, trimmer, logger)

	runSinglePass(t, svc, trimmer.trimmed)

	wantDeleted := []deletedSession{{agentID: testAgentID, sessID: "sess-old"}}
	if !reflect.DeepEqual(sessRepo.deleted, wantDeleted) {
		t.Fatalf("deleted sessions %v, want %v", sessRepo.deleted, wantDeleted)
	}
	if got := sessRepo.headersCalls(testAgentID); got != 1 {
		t.Fatalf("expected a single cleanup pass, got %d", got)
	}
	if got := handler.count(slog.LevelError); got != 0 {
		t.Fatalf("expected no errors during cleanup, got %d", got)
	}
	if got := trimmer.trimmedCount(); got != 1 {
		t.Fatalf("expected log trim invoked once, got %d", got)
	}
	if got := trimmer.maxLines(); got != testMaxLogLines {
		t.Fatalf("expected log trim with %d lines, got %d", testMaxLogLines, got)
	}
}

func TestRun_LogsSessionsCleanerError(t *testing.T) {
	logger, handler := newStubLogger()
	agentRepo := newMockAgentRepo()
	agentRepo.allErr = errors.New("storage failure")
	trimmer := &mockLogTrimmer{trimmed: make(chan struct{}, 4)}

	svc, err := cleanup.NewCleanUpService(cleanup.CleanUpConfig{
		SessRetention:   testRetention,
		CleanUpInterval: testInterval,
		MaxLogLines:     testMaxLogLines,
	}, cleanup.NewSessionsCleaner(agentRepo, &mockSessionsRepo{}, logger), trimmer, logger)
	if err != nil {
		t.Fatalf("create cleanup service: %v", err)
	}

	runSinglePass(t, svc, trimmer.trimmed)

	if got := handler.count(slog.LevelError); got == 0 {
		t.Fatal("expected sessions cleaner error to be logged")
	}
	if got := trimmer.trimmedCount(); got != 1 {
		t.Fatalf("expected log trim to run despite sessions error, got %d calls", got)
	}
}

func TestRun_LogsLogTrimError(t *testing.T) {
	logger, handler := newStubLogger()
	sessRepo := &mockSessionsRepo{}
	trimmer := &mockLogTrimmer{trimErr: errors.New("disk full"), trimmed: make(chan struct{}, 4)}

	svc, err := cleanup.NewCleanUpService(cleanup.CleanUpConfig{
		SessRetention:   testRetention,
		CleanUpInterval: testInterval,
		MaxLogLines:     testMaxLogLines,
	}, cleanup.NewSessionsCleaner(
		newMockAgentRepo(agent.NewAgent(testAgentID, "", "", "", nil, false)),
		sessRepo,
		logger,
	), trimmer, logger)
	if err != nil {
		t.Fatalf("create cleanup service: %v", err)
	}

	runSinglePass(t, svc, trimmer.trimmed)

	if got := handler.count(slog.LevelError); got == 0 {
		t.Fatal("expected log trim error to be logged")
	}
	if got := sessRepo.headersCalls(testAgentID); got != 1 {
		t.Fatalf("expected sessions cleaner to run despite trim error, got %d calls", got)
	}
}
