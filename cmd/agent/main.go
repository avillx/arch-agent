package main

import (
	"arch-agent/internal/api"
	"arch-agent/internal/logging"
	"arch-agent/internal/server"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const ShutdownTimeoutDuration = 10 * time.Second

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		slog.Error("system run", "error", err)
	}
}

func run(ctx context.Context) error {

	ctx, stop := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// ENV logging
	logIndented := getEnv("AGENT_LOG_INDENTED", false, strconv.ParseBool)
	logSource := getEnv("AGENT_LOG_SOURCE", false, strconv.ParseBool)
	logLevel := getEnv("AGENT_LOG_LEVEL", slog.LevelWarn, logging.ToLogLevel)
	logJSON := getEnv("AGENT_LOG_JSON", false, strconv.ParseBool)

	// ENV server config
	port := strconv.Itoa(getEnv("AGENT_PORT", 8080, strconv.Atoi))
	datapath := getEnv("AGENT_DATAPATH", "./agent-data", parseString)
	cleanupInterval := getEnv("AGENT_CLEAN_UP_INTERVAL", 12*time.Hour, time.ParseDuration)
	sessRetention := getEnv("AGENT_SESSION_RETENTION", 240*time.Hour, time.ParseDuration)
	maxLogLines := getEnv("AGENT_MAX_LOGLINES", 1000, strconv.Atoi)
	agentShell := getEnv("AGENT_SHELL", "", parseString)
	agentShellAttributes := strings.Fields(getEnv("AGENT_SHELL_ATTRIBUTES", "", parseString))

	// Logging
	logHandler := logging.NewHandler(logging.LoggerConfig{
		LogJSON:   logJSON,
		Level:     logLevel,
		AddSource: logSource,
		Indented:  logIndented,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	// App composing
	srv, err := server.New(ctx,
		datapath, maxLogLines, sessRetention, cleanupInterval, agentShell, agentShellAttributes, logger,
	)
	if err != nil {
		return err
	}

	httpServer := http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: api.NewHTTPHandler(logger, srv),
	}

	slog.Warn("start system")

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server", "error", err)
		}
		stop()
		slog.Warn("system shutdown")
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeoutDuration)
	defer cancel()

	return httpServer.Shutdown(shutdownCtx)
}

func getEnv[T any](key string, def T, parse func(string) (T, error)) T {
	raw, ok := os.LookupEnv(key)
	if !ok {
		slog.Info("Environment variable is not specified, default applied",
			"env", key, "default", def)
		return def
	}

	v, err := parse(raw)
	if err != nil {
		slog.Error("Environment variable can't be parsed, default applied",
			"env", key, "value", raw, "default", def, "error", err)
		return def
	}
	return v
}

func parseString(s string) (string, error) { return s, nil }
func parseInt64(s string) (int64, error)   { return strconv.ParseInt(s, 10, 64) }
