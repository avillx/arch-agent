package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
)

const flushInterval = 2 * time.Minute

type sessionKey struct {
	agentID   agent.ID
	sessionID session.ID
}

type interaction struct {
	mu        sync.Mutex
	agentID   agent.ID
	sessionID session.ID
	activity  string
	tokens    int64
	timer     *time.Timer
}

type Observer struct {
	interactions map[sessionKey]*interaction
	mu           sync.RWMutex

	model    agent.Model
	repo     agent.ActivityRepo
	tokenMax int64
}

func NewObserver(m agent.Model, repo agent.ActivityRepo) *Observer {
	return &Observer{
		model:        m,
		repo:         repo,
		tokenMax:     m.ContextLimit(),
		interactions: map[sessionKey]*interaction{},
	}
}

func (o *Observer) Intercept(ctx context.Context, additionalMessages []agent.Message, agentID agent.ID, sessID session.ID, evCh chan Event) chan Event {
	key := sessionKey{agentID, sessID}

	o.mu.Lock()
	inter, ok := o.interactions[key]
	if !ok {
		inter = o.createInteraction(key)
		o.interactions[key] = inter
	}
	for _, m := range additionalMessages {
		inter.activity += m.String()
	}
	o.mu.Unlock()

	sinkCh := make(chan Event, 16)

	go func() {
		reader := EventReader{
			OnEvent: func(ev Event) {
				evCh <- ev
				// case <-ctx.Done():
				// 	slog.Warn("observer: evCh full or closed, dropping event", "agent", agentID, "session", sessID)

			},
			OnComplete: func(_ agent.ID, _ session.ID, c *agent.Completion) {
				inter.mu.Lock()

				// tool call eliminated as unneccecary for log
				inter.activity += agent.NewAgentMessage(c.Content, nil).String()
				inter.mu.Unlock()

				if shouldCompact(c.InputTokens, c.CompletionTokens, o.model.ContextLimit()) {
					if err := o.flush(ctx, key); err != nil {
						slog.Error("observer", "error", err)
					}
				}
			},
		}

		reader.Read(sinkCh)
		close(evCh)
	}()

	return sinkCh
}

func (o *Observer) createInteraction(key sessionKey) *interaction {

	inter := &interaction{
		agentID:   key.agentID,
		sessionID: key.sessionID,
	}

	inter.timer = time.AfterFunc(flushInterval, func() {
		if err := o.flush(context.Background(), key); err != nil {
			slog.Error("observer", "error", err)
		}
	})

	return inter
}

func (o *Observer) flush(ctx context.Context, key sessionKey) error {
	o.mu.Lock()
	inter, ok := o.interactions[key]
	if !ok {
		o.mu.Unlock()
		return types.ErrIsNotExist
	}
	inter.timer.Stop()
	activity := inter.activity
	delete(o.interactions, key)
	o.mu.Unlock()

	completion, err := o.model.Complete(ctx, nil, []agent.Message{
		agent.NewSystemMessage(prompt.ReportSystem()),
		agent.NewUserMessage(
			fmt.Sprintf(
				"%s\n\n<transcript>%s</transcript>",
				prompt.ReportRequest(),
				activity,
			),
		),
	})
	if err != nil {
		return err
	}

	return o.repo.Log(key.agentID, agent.ActivityRecord{
		Content: completion.Content,
		Stamp:   time.Now(),
	})
}
