package cleanup_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/cleanup"
	"arch-agent/internal/session"
	"errors"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"
)

func newCleaner(sessRepo *mockSessionsRepo, agentRepo agent.Repo) *cleanup.SessionsCleaner {
	if agentRepo == nil {
		agentRepo = newMockAgentRepo(agent.NewAgent(testAgentID, "", "", "", nil, false))
	}
	return cleanup.NewSessionsCleaner(agentRepo, sessRepo, slog.New(slog.DiscardHandler))
}

func TestClean_DeletesDeprecatedSessions(t *testing.T) {
	sessRepo := &mockSessionsRepo{
		headersByAgent: map[agent.ID][]session.SessionHeader{
			testAgentID: {
				oldHeader("sess-old", time.Now().Add(-24*time.Hour)),
				oldHeader("sess-new", time.Now()),
			},
		},
	}
	cleaner := newCleaner(sessRepo, nil)

	if err := cleaner.Clean(testRetention); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []deletedSession{{agentID: testAgentID, sessID: "sess-old"}}
	if !reflect.DeepEqual(sessRepo.deleted, want) {
		t.Fatalf("deleted sessions %v, want %v", sessRepo.deleted, want)
	}
}

func TestClean_KeepsRecentSessions(t *testing.T) {
	sessRepo := &mockSessionsRepo{
		headersByAgent: map[agent.ID][]session.SessionHeader{
			testAgentID: {
				oldHeader("s1", time.Now()),
				oldHeader("s2", time.Now().Add(-time.Hour)),
			},
		},
	}
	cleaner := newCleaner(sessRepo, nil)

	if err := cleaner.Clean(testRetention); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessRepo.deleted) != 0 {
		t.Fatalf("expected no deletions, got %v", sessRepo.deleted)
	}
}

func TestClean_DeletesBrokenSessions(t *testing.T) {
	sessRepo := &mockSessionsRepo{
		headersErr: map[agent.ID]error{
			testAgentID: session.NewErrBrokenHeaders(
				session.NewErrBrokenHeader("broken-1", testAgentID, errors.New("cause")),
				session.NewErrBrokenHeader("broken-2", testAgentID, errors.New("cause")),
			),
		},
	}
	cleaner := newCleaner(sessRepo, nil)

	if err := cleaner.Clean(testRetention); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []deletedSession{
		{agentID: testAgentID, sessID: "broken-1"},
		{agentID: testAgentID, sessID: "broken-2"},
	}
	if !reflect.DeepEqual(sessRepo.deleted, want) {
		t.Fatalf("deleted sessions %v, want %v", sessRepo.deleted, want)
	}
}

func TestClean_SkipsAgentOnHeadersError(t *testing.T) {
	sessRepo := &mockSessionsRepo{
		headersErr: map[agent.ID]error{testAgentID: errors.New("read failure")},
	}
	cleaner := newCleaner(sessRepo, nil)

	if err := cleaner.Clean(testRetention); err != nil {
		t.Fatalf("headers error must not fail cleanup: %v", err)
	}

	if len(sessRepo.deleted) != 0 {
		t.Fatalf("expected no deletions, got %v", sessRepo.deleted)
	}
}

func TestClean_PropagatesAgentRepoError(t *testing.T) {
	agentRepo := newMockAgentRepo()
	agentRepo.allErr = errors.New("storage failure")
	cleaner := cleanup.NewSessionsCleaner(agentRepo, &mockSessionsRepo{}, slog.New(slog.DiscardHandler))

	err := cleaner.Clean(testRetention)
	if err == nil {
		t.Fatal("expected error from agent repo")
	}
	if !errors.Is(err, agentRepo.allErr) {
		t.Fatalf("expected original error, got %v", err)
	}
}

func TestClean_ContinuesOnDeleteError(t *testing.T) {
	old := time.Now().Add(-24 * time.Hour)
	sessRepo := &mockSessionsRepo{
		headersByAgent: map[agent.ID][]session.SessionHeader{
			testAgentID: {
				oldHeader("s1", old),
				oldHeader("s2", old),
			},
		},
		deleteErr: errors.New("disk full"),
	}
	cleaner := newCleaner(sessRepo, nil)

	if err := cleaner.Clean(testRetention); err != nil {
		t.Fatalf("delete error must not abort cleanup: %v", err)
	}

	if got := sessRepo.deleteAttemptCount(); got != 2 {
		t.Fatalf("expected both sessions attempted for deletion, got %d", got)
	}
	if len(sessRepo.deleted) != 0 {
		t.Fatalf("expected no successful deletions, got %v", sessRepo.deleted)
	}
}

func TestClean_ProcessesAllAgents(t *testing.T) {
	agent2 := agent.ID("0")
	old := time.Now().Add(-24 * time.Hour)
	sessRepo := &mockSessionsRepo{
		headersByAgent: map[agent.ID][]session.SessionHeader{
			testAgentID: {oldHeader("a1", old)},
			agent2:      {oldHeader("b1", old), oldHeader("b2", old)},
		},
	}
	agents := []agent.Agent{
		agent.NewAgent(testAgentID, "", "", "", nil, false),
		agent.NewAgent(agent2, "", "", "", nil, false),
	}
	cleaner := cleanup.NewSessionsCleaner(newMockAgentRepo(agents...), sessRepo, slog.New(slog.DiscardHandler))

	if err := cleaner.Clean(testRetention); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []deletedSession{
		{agentID: testAgentID, sessID: "a1"},
		{agentID: agent2, sessID: "b1"},
		{agentID: agent2, sessID: "b2"},
	}
	if !reflect.DeepEqual(sessRepo.deleted, want) {
		t.Fatalf("deleted sessions %v, want %v", sessRepo.deleted, want)
	}
}

func TestClean_IsConcurrencySafe(t *testing.T) {
	old := time.Now().Add(-24 * time.Hour)
	sessRepo := &mockSessionsRepo{
		headersByAgent: map[agent.ID][]session.SessionHeader{
			testAgentID: {oldHeader("s1", old), oldHeader("s2", old)},
		},
	}
	cleaner := newCleaner(sessRepo, nil)

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cleaner.Clean(testRetention); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := sessRepo.deleteAttemptCount(); got != 4 {
		t.Fatalf("expected 4 delete attempts (2 passes x 2 sessions), got %d", got)
	}
}
