package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
)

func Set(pretty bool, level slog.Level) {
	switch {
	case pretty:
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
		Level:     slog.LevelDebug,
		AddSource: true,
	}
	textHandler := slog.NewJSONHandler(&prettyWriter{os.Stdout}, opt)
	return slog.New(textHandler)
}
