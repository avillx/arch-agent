package subagent

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
)

type stubChatExecutor struct {
	events []runtime.Event
}

func (e *stubChatExecutor) Chat(_ context.Context, r chat.Request) error {
	for _, ev := range e.events {
		r.Sink <- ev
	}

	return nil
}

type stubSessionsRepo struct {
	savedAgentIDs []agent.ID
}

func (r *stubSessionsRepo) Session(agent.ID, session.ID) (session.Session, error) {
	return nil, nil
}

func (r *stubSessionsRepo) Save(agentID agent.ID, _ session.Session) error {
	r.savedAgentIDs = append(r.savedAgentIDs, agentID)
	return nil
}

func (r *stubSessionsRepo) Delete(agent.ID, session.ID) error {
	return nil
}

func (r *stubSessionsRepo) Headers(agent.ID) ([]session.SessionHeader, error) {
	return nil, nil
}

type stubUUID struct {
	id string
}

func (u stubUUID) New() string {
	return u.id
}

func newTestService(t *testing.T, chatExec chat.ChatExecutor, repo session.SessionsRepo) *Service {
	t.Helper()

	sessService := session.NewService(
		repo,
		stubUUID{id: "session-1"},
		slog.New(slog.DiscardHandler),
	)

	return NewService(
		chatExec,
		sessService,
		slog.New(slog.DiscardHandler),
	)
}

func TestCall_HappyPath(t *testing.T) {
	repo := &stubSessionsRepo{}
	chatExec := &stubChatExecutor{
		events: []runtime.Event{
			runtime.NewCompleteEvent(&agent.Completion{Content: "done", Done: true}),
		},
	}

	svc := newTestService(t, chatExec, repo)

	got, err := svc.Call(context.Background(), agent.ID("sub-agent"), "please do")
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if got != "done" {
		t.Fatalf("Call() = %q, want %q", got, "done")
	}
	if len(repo.savedAgentIDs) != 1 || repo.savedAgentIDs[0] != agent.ID("sub-agent") {
		t.Fatalf("expected one session for sub-agent, got %v", repo.savedAgentIDs)
	}
}

func TestCall_CallStackOverflow(t *testing.T) {
	repo := &stubSessionsRepo{}
	svc := newTestService(t, &stubChatExecutor{}, repo)

	ctx := context.Background()
	for range maxSubAgentDepth {
		ctx, _ = subAgentCallStack(ctx, subAgentCall{subagent: agent.ID("nested")})
	}

	got, err := svc.Call(ctx, agent.ID("sub-agent"), "please do")
	if !errors.Is(err, ErrCallStackOverflow) {
		t.Fatalf("Call() error = %v, want ErrCallStackOverflow", err)
	}
	if got != "" {
		t.Fatalf("Call() = %q, want empty result", got)
	}
	if len(repo.savedAgentIDs) != 0 {
		t.Fatalf("expected no session created, got %v", repo.savedAgentIDs)
	}
}
