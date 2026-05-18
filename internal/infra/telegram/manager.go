package telegram

import (
	service "arch-agent/internal/app"
	"arch-agent/internal/domain/agent"
	"context"
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

func (o *BotOrchestrator) Get(agentID agent.ID) *Bot {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.bots[string(agentID)]
}

func (o *BotOrchestrator) WireApp(app *service.App) {
	for _, b := range o.bots {
		b.WireApp(app)
	}
}
