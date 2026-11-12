package sentinel

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
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
	watcher *fsnotify.Watcher
}

func New(cwd string, logger *slog.Logger, opts ...Option) (*Sentinel, error) {

	cwd = filepath.Clean(cwd)

	s := &Sentinel{
		cwd:     cwd,
		actions: map[string]*debouncedAction{},
		logger:  logger.WithGroup("file_sentinel"),
	}

	for _, opt := range opts {
		opt(s)
	}

	watcher, err := fsnotify.NewBufferedWatcher(1)
	if err != nil {
		return nil, err
	}

	if err := watcher.Add(cwd); err != nil {
		return nil, err
	}

	s.watcher = watcher

	return s, nil
}

func (s *Sentinel) Run(ctx context.Context) {

	defer s.watcher.Close()

	for {
		select {
		case ev, ok := <-s.watcher.Events:
			if !ok {
				s.logger.Warn("watcher closed")
				return
			}

			s.mu.RLock()
			if a, ok := s.actions[ev.Name]; ok {
				go a.Do(ctx, ev)
			}
			s.mu.RUnlock()

		case err, ok := <-s.watcher.Errors:
			if !ok {
				s.logger.Warn("watcher closed")
				return
			}
			s.logger.Error("can't backup new data", "error", err)

		case <-ctx.Done():
			return
		}
	}
}

func WithWatch(p string, a Action) Option {
	return func(s *Sentinel) {
		s.mu.Lock()
		defer s.mu.Unlock()

		p = filepath.Join(s.cwd, p)

		s.actions[p] = &debouncedAction{
			action: a,
			logger: s.logger.With("path", p),
		}
	}
}
