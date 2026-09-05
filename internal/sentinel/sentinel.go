package sentinel

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounce = 300 * time.Millisecond

var ErrClosedWatcher = errors.New("fsnotify watcher closed")

type Action func(ctx context.Context, ev fsnotify.Event) error

type debouncedAction struct {
	dropUpdates   bool
	debounceTimer *time.Timer
	debounceMu    sync.Mutex
	action        Action
	logger        *slog.Logger
}

func (a *debouncedAction) Do(ctx context.Context, ev fsnotify.Event) {
	a.debounceMu.Lock()
	defer a.debounceMu.Unlock()

	if a.dropUpdates {
		return
	}

	a.dropUpdates = true
	a.logger.Debug("update detected, actions started")

	a.debounceTimer = time.AfterFunc(debounce, func() {
		if err := a.action(ctx, ev); err != nil {
			a.logger.Error("action failed", "error", err)
		}

		a.debounceMu.Lock()
		defer a.debounceMu.Unlock()

		a.dropUpdates = false
	})
}

type Option func(*Sentinel)

type Sentinel struct {
	cwd     string
	logger  *slog.Logger
	actions map[string]*debouncedAction
	mu      sync.RWMutex
}

func New(cwd string, logger *slog.Logger, opts ...Option) *Sentinel {
	s := &Sentinel{
		cwd:     cwd,
		actions: map[string]*debouncedAction{},
		logger:  logger.WithGroup("file_sentinel"),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Sentinel) Run(ctx context.Context) error {
	watcher, err := fsnotify.NewBufferedWatcher(1)
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(s.cwd); err != nil {
		return err
	}

	for {
		select {
		case ev, ok := <-watcher.Events:
			if !ok {
				return ErrClosedWatcher
			}

			// TODO: prefix + ev.Name event name?
			s.mu.RLock()
			if a, ok := s.actions[ev.Name]; ok {
				go a.Do(ctx, ev)
			}
			s.mu.RUnlock()

		case err, ok := <-watcher.Errors:
			if !ok {
				return ErrClosedWatcher
			}
			s.logger.Error("can't backup new data", "error", err)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func WithWatch(pathPrefix string, a Action) Option {
	return func(s *Sentinel) {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.actions[pathPrefix] = &debouncedAction{
			action: a,
			logger: s.logger.With("path", pathPrefix),
		}
	}
}
