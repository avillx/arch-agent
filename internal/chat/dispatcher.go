package chat

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"context"
	"log/slog"
	"sync"
)

var _ ChatExecutor = (*Dispatcher)(nil)

type sessionKey struct {
	AgentID   agent.ID
	SessionID session.ID
}

type requestProcessing struct {
	cancel context.CancelFunc
	done   chan struct{}
}

type Dispatcher struct {
	*Service
	processes map[sessionKey]*requestProcessing
	mu        sync.Mutex
}

func NewDispatcher(s *Service) *Dispatcher {
	return &Dispatcher{
		Service:   s,
		processes: map[sessionKey]*requestProcessing{},
	}
}

func (d *Dispatcher) Chat(ctx context.Context, r Request) error {
	key := sessionKey{
		AgentID:   r.AgentID,
		SessionID: r.SessionID,
	}

	ctx, cancel := context.WithCancel(ctx)
	p := &requestProcessing{
		cancel: cancel,
		done:   make(chan struct{}),
	}

	if prev := d.swap(key, p); prev != nil {
		prev.cancel()
		<-prev.done
	}

	defer func() {
		cancel()
		close(p.done)
		d.remove(key, p)
	}()

	return d.Service.Chat(ctx, r)
}

func (d *Dispatcher) Interrupt(sessID session.ID, agentID agent.ID) {
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

func (d *Dispatcher) swap(key sessionKey, p *requestProcessing) *requestProcessing {
	d.mu.Lock()
	defer d.mu.Unlock()

	prev := d.processes[key]
	d.processes[key] = p
	return prev
}

func (d *Dispatcher) remove(key sessionKey, p *requestProcessing) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.processes[key] == p {
		delete(d.processes, key)
	}
}
