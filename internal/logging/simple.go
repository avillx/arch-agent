package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

type simpleHandler struct {
	w      io.Writer
	level  slog.Level
	attrs  []slog.Attr
	groups []string

	mu sync.Mutex
}

func NewSimpleHandler(w io.Writer, level slog.Level) slog.Handler {
	return &simpleHandler{
		w:      w,
		level:  level,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

func (h *simpleHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level
}

func (h *simpleHandler) Handle(_ context.Context, r slog.Record) error {

	h.mu.Lock()
	defer h.mu.Unlock()

	// base message
	msg := []string{
		fmt.Sprintf("[%s]", simpleTag(r.Level)),
		r.Time.Format("2006-01-02 15:04:05"),
	}

	// gather groups
	if len(h.groups) > 0 {
		msg = append(msg, strings.Join(h.groups, "."))
	}

	// gather attributes
	// TODO: better support for complex slog types
	for _, a := range h.attrs {
		msg = append(msg, fmt.Sprintf("%s=%v", a.Key, a.Value))
	}
	r.Attrs(func(a slog.Attr) bool {
		msg = append(msg, fmt.Sprintf("%s=%v", a.Key, a.Value))
		return true
	})

	// add message
	msg = append(msg, r.Message)

	// flush
	_, err := fmt.Fprintln(h.w, strings.Join(msg, " "))
	return err
}

func (h *simpleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &simpleHandler{
		w:      h.w,
		level:  h.level,
		attrs:  append(h.attrs, attrs...),
		groups: h.groups,
	}
}

func (h *simpleHandler) WithGroup(g string) slog.Handler {
	return &simpleHandler{
		w:      h.w,
		level:  h.level,
		attrs:  h.attrs,
		groups: append(h.groups, g),
	}
}

func simpleTag(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARN"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}
