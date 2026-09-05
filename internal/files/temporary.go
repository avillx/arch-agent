package files

import (
	"arch-agent/internal/sentinel"
	"arch-agent/internal/types"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const tmpLifeTime = 10 * time.Minute
const TMPDir = "tmp"

type TemporaryFiles struct {
	timers map[string]*time.Timer
	fs     *FileSystem

	logger *slog.Logger
	mu     sync.Mutex
}

func NewTemporaryFiles(fs *FileSystem, logger *slog.Logger) (*TemporaryFiles, error) {
	f := &TemporaryFiles{
		timers: map[string]*time.Timer{},
		fs:     fs,
		logger: logger.WithGroup("tmp"),
	}

	if err := f.stageEntry(); err != nil {
		return nil, err
	}

	return f, nil
}

func (f *TemporaryFiles) Reload(_ context.Context) error {
	return f.stageEntry()
}

func (f *TemporaryFiles) Run(ctx context.Context) {

	<-ctx.Done()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, t := range f.timers {
		t.Stop()
	}
}

func (f *TemporaryFiles) stageEntry() error {

	entry, err := f.fs.ReadDir(TMPDir)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			// create if has no
			return f.fs.MkdirAll(TMPDir)
		}
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.releaseUnexisted(entry)
	f.stageNew(entry)

	return nil
}

func (f *TemporaryFiles) stageNew(entry []os.DirEntry) {
	for _, e := range entry {
		entryPath := resolveTmpEntryPath(e.Name())

		// add new
		if _, ok := f.timers[entryPath]; !ok {
			f.watchFor(entryPath)
		}
	}
}

func (f *TemporaryFiles) watchFor(p string) {

	f.timers[p] = time.AfterFunc(tmpLifeTime, func() {
		f.mu.Lock()
		defer f.mu.Unlock()

		logger := f.logger.With("path", p)
		logger.Info("deletion")
		if err := f.fs.DeleteAll(p); err != nil {
			if !errors.Is(err, types.ErrIsNotExist) {
				logger.Error("deletion", "error", err)
			}
		}

		if t, ok := f.timers[p]; ok {
			t.Stop()
			delete(f.timers, p)
		}
	})

	f.logger.Info("element deletion planned", "element", p, "TTL (minutes)", (tmpLifeTime / time.Minute))
}

func (f *TemporaryFiles) releaseUnexisted(existed []os.DirEntry) {

	existMap := make(map[string]struct{}, len(existed))
	for _, e := range existed {
		existMap[resolveTmpEntryPath(e.Name())] = struct{}{}
	}

	for k, t := range f.timers {
		if _, ok := existMap[k]; !ok {
			t.Stop()
			delete(f.timers, k)
		}
	}
}

func resolveTmpEntryPath(e string) string {
	return filepath.Join(TMPDir, e)
}

// tmpSent
func NewTMPDetector(tmpFiles *TemporaryFiles) sentinel.Action {
	return func(ctx context.Context, ev fsnotify.Event) error {
		return tmpFiles.Reload(ctx)
	}
}
