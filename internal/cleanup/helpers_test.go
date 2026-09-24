package cleanup_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

const (
	testRetention   = 10 * time.Hour
	testInterval    = 5 * time.Hour
	testMaxLogLines = 5
	testAgentID     = agent.ID("agi-1")
)

var errNotImplemented = errors.New("not implemented in test")

type mockAgentRepo struct {
	agents []agent.Agent
	allErr error
}

func newMockAgentRepo(agents ...agent.Agent) *mockAgentRepo {
	return &mockAgentRepo{agents: agents}
}

func (m *mockAgentRepo) All() ([]agent.Agent, error) {
	return m.agents, m.allErr
}

func (m *mockAgentRepo) Get(agent.ID) (agent.Agent, error) {
	return nil, errNotImplemented
}

func (m *mockAgentRepo) Save(agent.Agent) error {
	return errNotImplemented
}

func (m *mockAgentRepo) Delete(agent.ID) error {
	return errNotImplemented
}

type deletedSession struct {
	agentID agent.ID
	sessID  session.ID
}

type mockSessionsRepo struct {
	headersByAgent map[agent.ID][]session.SessionHeader
	headersErr     map[agent.ID]error
	deleteErr      error

	mu                sync.Mutex
	deleteAttempts    []deletedSession
	deleted           []deletedSession
	headersCallsCount map[agent.ID]int
}

func (m *mockSessionsRepo) Session(agent.ID, session.ID) (session.Session, error) {
	return nil, errNotImplemented
}

func (m *mockSessionsRepo) Save(agent.ID, session.Session) error {
	return errNotImplemented
}

func (m *mockSessionsRepo) Delete(agentID agent.ID, sessID session.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteAttempts = append(m.deleteAttempts, deletedSession{agentID: agentID, sessID: sessID})
	if m.deleteErr != nil {
		return m.deleteErr
	}

	m.deleted = append(m.deleted, deletedSession{agentID: agentID, sessID: sessID})
	return nil
}

func (m *mockSessionsRepo) Headers(agentID agent.ID) ([]session.SessionHeader, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.headersCallsCount == nil {
		m.headersCallsCount = map[agent.ID]int{}
	}
	m.headersCallsCount[agentID]++

	if err, ok := m.headersErr[agentID]; ok && err != nil {
		return nil, err
	}

	headers, ok := m.headersByAgent[agentID]
	if !ok {
		return nil, nil
	}
	return headers, nil
}

func (m *mockSessionsRepo) deleteAttemptCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.deleteAttempts)
}

func (m *mockSessionsRepo) headersCalls(agentID agent.ID) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count, ok := m.headersCallsCount[agentID]
	if !ok {
		return 0
	}
	return count
}

func oldHeader(id session.ID, updatedAt time.Time) session.SessionHeader {
	return session.NewHeader(id, 0, 0, updatedAt, updatedAt, nil)
}

type mockLogTrimmer struct {
	mu           sync.Mutex
	calls        int
	lastMaxLines int
	trimErr      error
	trimmed      chan struct{}
}

func (m *mockLogTrimmer) Trim(maxLines int) error {
	m.mu.Lock()
	m.calls++
	m.lastMaxLines = maxLines
	m.mu.Unlock()

	if m.trimmed != nil {
		select {
		case m.trimmed <- struct{}{}:
		default:
		}
	}

	return m.trimErr
}

func (m *mockLogTrimmer) trimmedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.calls
}

func (m *mockLogTrimmer) maxLines() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.lastMaxLines
}

type stubHandler struct {
	mu      sync.Mutex
	records map[slog.Level]int
}

func newStubLogger() (*slog.Logger, *stubHandler) {
	h := &stubHandler{records: map[slog.Level]int{}}
	return slog.New(h), h
}

func (h *stubHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *stubHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.records[r.Level]++
	return nil
}

func (h *stubHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *stubHandler) WithGroup(_ string) slog.Handler {
	return h
}

func (h *stubHandler) count(level slog.Level) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.records[level]
}
