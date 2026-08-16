package chat

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"context"
	"log/slog"
	"sync"
)

type sessionKey struct {
	AgentID   agent.ID
	SessionID session.ID
}

type requestProcessing struct {
	cancel context.CancelFunc
	done   chan struct{}
}

type Service struct {
	executor  *executor
	processes map[sessionKey]*requestProcessing
	mu        sync.Mutex
}

func NewService(e *executor) *Service {
	return &Service{
		executor:  e,
		processes: map[sessionKey]*requestProcessing{},
	}
}

func (s *Service) Chat(ctx context.Context, r Request) error {

	if err := r.validate(); err != nil {
		return err
	}

	return s.dispatch(ctx, r)
}

func (d *Service) Interrupt(sessID session.ID, agentID agent.ID) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := sessionKey{
		AgentID:   agentID,
		SessionID: sessID,
	}

	if prev := d.processes[key]; prev != nil {
		prev.cancel()
		<-prev.done
		slog.Info("chat service: session interrupted", "session", key.SessionID, "agent", key.AgentID)
	}
}

func (d *Service) swap(key sessionKey, p *requestProcessing) *requestProcessing {
	d.mu.Lock()
	defer d.mu.Unlock()

	prev := d.processes[key]
	d.processes[key] = p
	return prev
}

func (d *Service) remove(key sessionKey, p *requestProcessing) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.processes[key] == p {
		delete(d.processes, key)
	}
}

func (s *Service) dispatch(ctx context.Context, r Request) error {
	key := sessionKey{
		AgentID:   r.AgentID,
		SessionID: r.SessionID,
	}

	ctx, cancel := context.WithCancel(ctx)
	p := &requestProcessing{
		cancel: cancel,
		done:   make(chan struct{}),
	}

	if prev := s.swap(key, p); prev != nil {
		prev.cancel()
		<-prev.done
	}

	defer func() {
		cancel()
		close(p.done)
		s.remove(key, p)
	}()

	return s.executor.chat(ctx, r)
}

type agentIDCTXKey struct{}
type sessionIDCTXKey struct{}

func withAgentID(ctx context.Context, id agent.ID) context.Context {
	return context.WithValue(ctx, agentIDCTXKey{}, id)
}
func AgentIDFromContext(ctx context.Context) (agent.ID, bool) {
	id, ok := ctx.Value(agentIDCTXKey{}).(agent.ID)
	return id, ok
}

func withSessionID(ctx context.Context, id session.ID) context.Context {
	return context.WithValue(ctx, sessionIDCTXKey{}, id)
}
func SessionIDFromContext(ctx context.Context) (session.ID, bool) {
	id, ok := ctx.Value(sessionIDCTXKey{}).(session.ID)
	return id, ok
}
