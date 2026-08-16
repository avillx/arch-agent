package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
)

type sessionKey struct {
	AgentID   agent.ID
	SessionID session.ID
}

const flushInterval = 2 * time.Minute

// concurent safe messages buffer
type messageBuffer struct {
	msgs []agent.Message
	mu   sync.Mutex
}

func newMessageBuffer() *messageBuffer {
	return &messageBuffer{
		msgs: []agent.Message{},
	}
}

func (b *messageBuffer) append(msgs []agent.Message) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.msgs = append(b.msgs, msgs...)
}

func (b *messageBuffer) transcript() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	var gatheredMsgs strings.Builder
	for _, m := range b.msgs {
		gatheredMsgs.WriteString(m.String())
	}

	return fmt.Sprintf(
		"%s\n\n<transcript>%s</transcript>",
		prompt.ReportRequest(),
		gatheredMsgs.String(),
	)
}

type Observer struct {
	messageBuffers map[sessionKey]*messageBuffer
	flushTimer     map[sessionKey]*time.Timer

	reporter *ActivityReporter
	repo     agent.ActivityRepo

	mu sync.RWMutex
}

func NewObserver(reprter *ActivityReporter, repo agent.ActivityRepo) *Observer {
	return &Observer{
		messageBuffers: map[sessionKey]*messageBuffer{},
		flushTimer:     map[sessionKey]*time.Timer{},
		reporter:       reprter,
		repo:           repo,
	}
}

func (o *Observer) getBuffer(key sessionKey) *messageBuffer {
	o.mu.Lock()
	defer o.mu.Unlock()

	buf, ok := o.messageBuffers[key]
	if !ok {
		buf = newMessageBuffer()
		timer := time.AfterFunc(flushInterval, func() {
			if err := o.flush(context.Background(), key); err != nil {
				slog.Error("observer", "error", err)
			}
		})
		o.flushTimer[key] = timer
		o.messageBuffers[key] = buf
	}

	return buf
}

// for both user and completion stuff
func (o *Observer) Commit(agentID agent.ID, sessID session.ID, msgs []agent.Message) {
	key := sessionKey{
		AgentID:   agentID,
		SessionID: sessID,
	}

	buf := o.getBuffer(key)
	buf.append(msgs)
}

func (o *Observer) releaseBuffer(key sessionKey) (*messageBuffer, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	buf, ok := o.messageBuffers[key]
	if !ok {
		return nil, types.ErrIsNotExist
	}

	delete(o.flushTimer, key)
	delete(o.messageBuffers, key)

	return buf, nil
}

func (o *Observer) flush(ctx context.Context, key sessionKey) error {

	buf, err := o.releaseBuffer(key)
	if err != nil {
		return err
	}

	extract, err := o.reporter.Extract(ctx, buf)
	if err != nil {
		return err
	}

	return o.repo.Log(
		key.AgentID,
		agent.ActivityRecord{
			Content: extract,
			Stamp:   time.Now(),
		},
	)
}

// transform messages in summary of important actions
type ActivityReporter struct {
	model agent.Model
}

func NewActivityReporter(model agent.Model) *ActivityReporter {
	return &ActivityReporter{
		model: model,
	}
}

func (r *ActivityReporter) Extract(ctx context.Context, buf *messageBuffer) (string, error) {

	msgs := []agent.Message{
		agent.NewSystemMessage(prompt.ReportSystem()),
		agent.NewUserMessage(buf.transcript()),
	}

	completion, err := r.model.Complete(ctx, nil, msgs)
	if err != nil {
		return "", err
	}

	return completion.Content, nil
}
