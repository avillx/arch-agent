package main

import (
	wire "arch-agent/internal"
	"arch-agent/internal/logging"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		slog.Error("system run", "error", err)
	}
	slog.Warn("system shutdown")
}

func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("bad envirement variable: %w", err)
	}

	// Logging has complex logic and is part of the business logic due to the
	// specifics of the project, so the logging config can't be set up in main.

	// App composing
	srv, err := wire.BuildServer(ctx, cfg)
	if err != nil {
		return err
	}

	httpServer := http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: srv,
	}

	slog.Warn("start system")

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	<-ctx.Done()

	return httpServer.Shutdown(context.Background())
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func LoadConfig() (wire.Config, error) {

	dataPath := getEnv("DATA_PATH", "./agent-data")
	port := getEnv("PORT", "8080")
	logIndented := getEnv("LOG_INDENTED", "false")
	logSource := getEnv("LOG_SOURCE", "false")
	logLevel := getEnv("LOG_LEVEL", "error")
	logJSON := getEnv("LOG_JSON", "false")
	cleanupInterval := getEnv("CLEAN_UP_INTERVAL", "12")
	sessRetention := getEnv("SESSION_RETENTION", "240")
	maxLogLines := getEnv("MAX_LOG_LINES", "1000")

	cleanupIntervalInt, err := strconv.ParseInt(cleanupInterval, 10, 64)
	if err != nil {
		return wire.Config{}, err
	}

	sessRetentionInt, err := strconv.ParseInt(sessRetention, 10, 64)
	if err != nil {
		return wire.Config{}, err
	}

	maxLogLinesInt, err := strconv.ParseInt(maxLogLines, 10, 64)
	if err != nil {
		return wire.Config{}, err
	}

	logSourceBool, err := strconv.ParseBool(logSource)
	if err != nil {
		return wire.Config{}, err
	}

	logIndentedBool, err := strconv.ParseBool(logIndented)
	if err != nil {
		return wire.Config{}, err
	}

	logLevelTyped, err := logging.ToLogLevel(logLevel)
	if err != nil {
		return wire.Config{}, err
	}

	logJSONBool, err := strconv.ParseBool(logJSON)
	if err != nil {
		return wire.Config{}, err
	}

	return wire.Config{
		DataPath:        dataPath,
		Port:            port,
		LogIndented:     logIndentedBool,
		LogSource:       logSourceBool,
		LogLevel:        logLevelTyped,
		LogJSON:         logJSONBool,
		SessRetention:   time.Duration(sessRetentionInt) * time.Hour,
		CleanUpInterval: time.Duration(cleanupIntervalInt) * time.Hour,
		MaxLogLines:     int(maxLogLinesInt),
	}, nil
}
