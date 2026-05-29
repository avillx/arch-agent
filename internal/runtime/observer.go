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
	tokens    int
	timer     *time.Timer
}

type Observer struct {
	interactions map[sessionKey]*interaction
	mu           sync.RWMutex

	model    agent.Model
	repo     agent.ActivityRepo
	counter  session.TokenCounter
	tokenMax int
}

func NewObserver(m agent.Model, repo agent.ActivityRepo, counter session.TokenCounter) *Observer {
	return &Observer{
		model:        m,
		repo:         repo,
		counter:      counter,
		tokenMax:     m.ContextLimit(),
		interactions: map[sessionKey]*interaction{},
	}
}

// Intercept подключается к потоку событий агента и наблюдает за активностью.
// Возвращает канал — прозрачный pass-through для вызывающего.
func (o *Observer) Intercept(ctx context.Context, request string, agentID agent.ID, sessID session.ID, evCh chan Event) chan Event {
	key := sessionKey{agentID, sessID}

	o.mu.Lock()
	inter, ok := o.interactions[key]
	if !ok {
		inter = o.createInteraction(key)
		o.interactions[key] = inter
	}
	inter.activity += agent.NewUserMessage(request).String()
	o.mu.Unlock()

	out := make(chan Event, 16)

	go func() {
		defer close(out)

		reader := EventReader{
			OnEvent: func(ev Event) {
				out <- ev
			},
			OnComplete: func(_ agent.ID, _ session.ID, c *agent.Completion) {
				inter.mu.Lock()
				inter.activity += agent.NewAgentMessage(c.Content, c.ToolCalls).String()
				inter.tokens += o.counter.Calc(c.Content)
				shouldFlush := inter.tokens >= o.tokenMax
				inter.mu.Unlock()

				if shouldFlush {
					if err := o.flush(ctx, key); err != nil {
						slog.Error("observer", "error", err)
					}
				}
			},
		}

		reader.Read(evCh)
	}()

	return out
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
