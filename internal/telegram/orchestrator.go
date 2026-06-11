package telegram

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/session"
	"context"
	"fmt"
	"sync"
)

type BotOrchestrator struct {
	bots map[string]*Bot
	mu   sync.RWMutex
}

func NewBotOrchestrator(cfgs ...BotConfig) (*BotOrchestrator, error) {

	o := &BotOrchestrator{
		bots: map[string]*Bot{},
	}

	for _, cfg := range cfgs {
		bot, err := NewBot(cfg)
		if err != nil {
			return nil, err
		}

		o.bots[cfg.Agent] = bot
	}

	return o, nil
}

func (o *BotOrchestrator) Run(ctx context.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, b := range o.bots {
		go b.Run(ctx)
	}
}

func (o *BotOrchestrator) Get(agentID agent.ID) (*Bot, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	bot, ok := o.bots[string(agentID)]
	if !ok {
		return nil, fmt.Errorf("bot for agent %s not found", agentID)
	}

	return bot, nil
}

func (o *BotOrchestrator) Wire(
	sessionSvc *session.Service,
	chatSvc *chat.Service,
	dreamer Dreamer,
) {
	for _, b := range o.bots {
		b.Wire(
			sessionSvc,
			chatSvc,
			dreamer,
		)
	}
}
