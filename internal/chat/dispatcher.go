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

type dispatcher struct {
	processes map[sessionKey]*requestProcessing
	mu        sync.Mutex
}

func (d *dispatcher) swap(key sessionKey, p *requestProcessing) *requestProcessing {
	d.mu.Lock()
	defer d.mu.Unlock()

	prev := d.processes[key]
	d.processes[key] = p
	return prev
}

func (d *dispatcher) remove(key sessionKey, p *requestProcessing) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.processes[key] == p {
		delete(d.processes, key)
	}
}

func (d *dispatcher) Interrupt(key sessionKey) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if prev := d.processes[key]; prev != nil {
		prev.cancel()
		<-prev.done
		slog.Warn("session interrupted", "session", key.SessionID, "agent", key.AgentID)
	}
}

func (d *dispatcher) Dispatch(ctx context.Context, r Request, handler func(ctx context.Context, r Request) error) error {
	key := sessionKey{AgentID: r.AgentID, SessionID: r.SessionID}

	ctx, cancel := context.WithCancel(ctx)
	p := &requestProcessing{cancel: cancel, done: make(chan struct{})}

	if prev := d.swap(key, p); prev != nil {
		prev.cancel()
		<-prev.done
	}

	defer func() {
		cancel()
		close(p.done)
		d.remove(key, p)
	}()

	return handler(ctx, r)
}
