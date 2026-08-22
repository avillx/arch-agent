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

type LoggerConfig struct {
	Level        slog.Level
	AddSource    bool
	Indented     bool
	JSON         bool
	AgentLogFile string
}

func NewLogger(cfg LoggerConfig) *slog.Logger {

	opt := &slog.HandlerOptions{
		Level:       cfg.Level,
		AddSource:   cfg.AddSource,
		ReplaceAttr: replaceAttrs,
	}

	// default
	var sink io.Writer = os.Stdout
	var handler slog.Handler = slog.NewTextHandler(sink, opt)

	// override
	if cfg.JSON {
		if cfg.Indented {
			sink = &prettyWriter{sink}
		}
		handler = slog.NewJSONHandler(sink, opt)
	}

	// add log for agent
	if cfg.AgentLogFile != "" {
		agentHandler := slog.NewTextHandler(
			newFileWriter(cfg.AgentLogFile),
			&slog.HandlerOptions{
				AddSource: false,
				Level:     slog.LevelInfo,
			},
		)
		handler = newMultiHandler([]slog.Handler{handler, agentHandler})
	}

	return slog.New(handler)
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

func replaceAttrs(groups []string, a slog.Attr) slog.Attr {
	switch v := a.Value.Any().(type) {
	// message
	case []agent.Message:
		attrs := []slog.Attr{}
		for i, m := range v {
			attrs = append(attrs, contentPartsToAtts(fmt.Sprintf("%d", i), m.Content()))
		}
		return slog.GroupAttrs(a.Key, attrs...)

	// contentPart
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
