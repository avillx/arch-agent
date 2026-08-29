package files

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

type Action func(context.Context) error

type Sentinel struct {
	fs          *FileSystem
	path        string
	action      Action
	logger      *slog.Logger
	dropUpdates bool

	debounceTimer *time.Timer
	debounceMu    sync.Mutex
}

func NewSentinel(fs *FileSystem, p string, a Action, logger *slog.Logger) (*Sentinel, error) {

	// fsnotify allows only existed path's
	// for resolve to abs
	p, err := fs.resolvePath(p)
	if err != nil {
		return nil, err
	}

	return &Sentinel{
		fs:     fs,
		path:   p,
		action: a,
		logger: logger.WithGroup("file_sentinel").With("path", p),
	}, nil
}

func (s *Sentinel) doAction(ctx context.Context) {

	s.debounceMu.Lock()
	defer s.debounceMu.Unlock()

	if s.dropUpdates {
		return
	}

	s.dropUpdates = true
	s.logger.Info("update detected, actions started")

	debounceFunc := func() {
		if err := s.action(ctx); err != nil {
			s.logger.Error("action failed", "error", err)
		}

		s.debounceMu.Lock()
		defer s.debounceMu.Unlock()
		s.dropUpdates = false
	}

	s.debounceTimer = time.AfterFunc(debounce, debounceFunc)
}

func (s *Sentinel) Run(ctx context.Context) error {
	watcher, err := fsnotify.NewBufferedWatcher(1)
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(s.path); err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return ErrClosedWatcher
			}
			if !event.Has(fsnotify.Write) {
				continue
			}

			s.doAction(ctx)

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
