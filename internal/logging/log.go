package logging

import (
	"arch-agent/internal/agent"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

func ToLogLevel(level string) (slog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level: %s", level)
	}
}

func Set(isPretty bool, level slog.Level) {
	switch {
	case isPretty:
		slog.SetDefault(NewPrettyLogger())
	default:
		slog.SetDefault(DefaultLogger())
	}

	slog.SetLogLoggerLevel(level)
}

// default logger
func DefaultLogger() *slog.Logger {
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	return slog.New(jsonHandler)
}

type prettyWriter struct {
	out io.Writer
}

func (w *prettyWriter) Write(p []byte) (int, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, p, "", "  "); err != nil {
		return w.out.Write(p)
	}
	buf.WriteByte('\n')
	return w.out.Write(buf.Bytes())
}

// logs to stdout pretty json's
func NewPrettyLogger() *slog.Logger {
	opt := &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		AddSource:   true,
		ReplaceAttr: replaceAttrs,
	}
	textHandler := slog.NewJSONHandler(&prettyWriter{os.Stdout}, opt)
	return slog.New(textHandler)
}

func replaceAttrs(groups []string, a slog.Attr) slog.Attr {
	switch v := a.Value.Any().(type) {
	case []agent.Message:
		attrs := []slog.Attr{}
		for i, m := range v {
			attrs = append(attrs, contentPartsToAtts(fmt.Sprintf("%d", i), m.Content()))
		}
		return slog.GroupAttrs(a.Key, attrs...)
	case []agent.ContentPart:
		return contentPartsToAtts(a.Key, v)
	default:
		return a
	}
}

func contentPartsToAtts(key string, parts []agent.ContentPart) slog.Attr {

	items := make([]any, len(parts))
	for i, p := range parts {
		items[i] = map[string]any{
			"Text":         p.Text,
			"ContainImage": p.ImageURL != "",
		}
	}
	return slog.Attr{
		Key:   key,
		Value: slog.AnyValue(items),
	}
}
