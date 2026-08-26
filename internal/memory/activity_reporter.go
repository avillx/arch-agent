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

const defaultFlushInterval = 120 // multiplies on seconds

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

type ActivityConfig struct {
	Enabled   bool   `toml:"enabled"`
	Interval  int64  `toml:"interval"`
	ModelName string `toml:"model"`
}

type ActivityConfigRepo interface {
	Save(ActivityConfig) error
	Load() (ActivityConfig, error)
}

type ActivityService struct {
	interval  int64
	modelName string
	enabled   bool
	model     agent.Model
	cfgMu     sync.RWMutex

	messageBuffers map[sessionKey]*messageBuffer
	flushTimer     map[sessionKey]*time.Timer
	mu             sync.RWMutex

	activityRepo agent.ActivityRepo
	cfgRepo      ActivityConfigRepo
	modelRepo    agent.ModelRegistry

	logger *slog.Logger
}

func NewActivityService(
	modelRepo agent.ModelRegistry,
	cfgRepo ActivityConfigRepo,
	activityRepo agent.ActivityRepo,
	logger *slog.Logger,
) *ActivityService {
	svc := &ActivityService{
		messageBuffers: map[sessionKey]*messageBuffer{},
		flushTimer:     map[sessionKey]*time.Timer{},
		logger:         logger.WithGroup("activity"),
		activityRepo:   activityRepo,
		interval:       defaultFlushInterval,
		cfgRepo:        cfgRepo,
		modelRepo:      modelRepo,
	}

	if err := svc.Reload(); err != nil {
		svc.logger.Error("service is not loaded", "error", err)
	}

	return svc
}

// for both user and completion stuff
func (o *ActivityService) Commit(agentID agent.ID, sessID session.ID, msgs []agent.Message) {
	if !o.Config().Enabled {
		// if sevice disabled then drop commits silently
		return
	}

	key := sessionKey{
		AgentID:   agentID,
		SessionID: sessID,
	}

	buf := o.getBuffer(key)
	buf.append(msgs)
}

func (o *ActivityService) releaseBuffer(key sessionKey) (*messageBuffer, error) {
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

func (o *ActivityService) getBuffer(key sessionKey) *messageBuffer {
	o.mu.Lock()
	defer o.mu.Unlock()

	buf, ok := o.messageBuffers[key]
	if !ok {
		buf = newMessageBuffer()
		interval := time.Duration(o.Config().Interval) * time.Second
		timer := time.AfterFunc(interval, func() {

			logger := o.logger.With("agent", key.AgentID, "session", key.SessionID)

			if !o.Config().Enabled {
				logger.Warn("not logged, service disabled")
				return
			}

			if err := o.flush(context.Background(), key); err != nil {
				logger.Error("not logged", "error", err)
				return
			}

			logger.Info("logged")
		})
		o.flushTimer[key] = timer
		o.messageBuffers[key] = buf
	}

	return buf
}

func (o *ActivityService) flush(ctx context.Context, key sessionKey) error {

	buf, err := o.releaseBuffer(key)
	if err != nil {
		return err
	}

	extract, err := o.extractActivity(ctx, buf)
	if err != nil {
		return err
	}

	return o.activityRepo.Log(
		key.AgentID,
		agent.ActivityRecord{
			Content: extract,
			Stamp:   time.Now(),
		},
	)
}

func (r *ActivityService) extractActivity(ctx context.Context, buf *messageBuffer) (string, error) {

	r.cfgMu.RLock()
	model := r.model
	r.cfgMu.RUnlock()

	if model == nil {
		return "", fmt.Errorf("model is not set")
	}

	msgs := []agent.Message{
		agent.NewSystemMessage(prompt.ReportSystem()),
		agent.NewUserMessage(buf.transcript()),
	}

	completion, err := model.Complete(ctx, nil, msgs)
	if err != nil {
		return "", err
	}

	return completion.Content, nil
}

func (s *ActivityService) Reload() error {
	cfg, err := s.cfgRepo.Load()
	if err != nil {
		return err
	}
	return s.SetConfig(cfg)
}

func (s *ActivityService) SetConfig(cfg ActivityConfig) error {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	model, err := s.modelRepo.Get(s.modelName)
	if err != nil {
		return err
	}
	s.model = model
	s.modelName = cfg.ModelName
	s.interval = cfg.Interval
	s.enabled = cfg.Enabled

	return nil
}

func (s *ActivityService) Config() ActivityConfig {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()

	return ActivityConfig{
		Interval:  s.interval,
		ModelName: s.modelName,
	}
}
