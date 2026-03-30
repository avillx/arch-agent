package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"os"
)

func NewCustomLogger() *slog.Logger {

	file, err := os.OpenFile(".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	textHandler := slog.NewTextHandler(file, nil)
	return slog.New(textHandler)
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

func NewConsoleLogger() *slog.Logger {
	opt := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}
	textHandler := slog.NewJSONHandler(&prettyWriter{os.Stdout}, opt)
	return slog.New(textHandler)
}
