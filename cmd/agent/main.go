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

type RunParameters struct {
	DataPath        string
	Port            string
	LogIndented     bool
	LogSource       bool
	LogLevel        slog.Level
	LogJSON         bool
	cleanupInterval time.Duration
	sessRetention   time.Duration
	maxLogLines     int
}

func NewRunParameters(
	dataPath string,
	port string,
	logIndented string,
	logSource string,
	logLevel string,
	logJSON string,
	cleanupInterval string,
	sessRetention string,
	maxLogLines string,
) (RunParameters, error) {

	if dataPath == "" {
		dataPath = "."
	}

	if port == "" {
		port = "8080"
	}

	if logJSON == "" {
		logJSON = "false"
	}

	if logIndented == "" {
		logIndented = "false"
	}

	if logSource == "" {
		logSource = "false"
	}

	if logLevel == "" {
		logLevel = "error"
	}

	if cleanupInterval == "" {
		cleanupInterval = "12"
	}
	if sessRetention == "" {
		sessRetention = "240"
	}
	if maxLogLines == "" {
		maxLogLines = "1000"
	}

	cleanupIntervalInt, err := strconv.ParseInt(cleanupInterval, 64, 10)
	if err != nil {
		return RunParameters{}, err
	}

	sessRetentionInt, err := strconv.ParseInt(sessRetention, 64, 10)
	if err != nil {
		return RunParameters{}, err
	}

	maxLogLinesInt, err := strconv.ParseInt(maxLogLines, 64, 10)
	if err != nil {
		return RunParameters{}, err
	}

	logSourceBool, err := strconv.ParseBool(logSource)
	if err != nil {
		return RunParameters{}, err
	}

	logIndentedBool, err := strconv.ParseBool(logIndented)
	if err != nil {
		return RunParameters{}, err
	}

	logLevelTyped, err := logging.ToLogLevel(logLevel)
	if err != nil {
		return RunParameters{}, err
	}

	logJSONBool, err := strconv.ParseBool(logJSON)
	if err != nil {
		return RunParameters{}, err
	}

	return RunParameters{
		DataPath:        dataPath,
		Port:            port,
		LogIndented:     logIndentedBool,
		LogSource:       logSourceBool,
		LogLevel:        logLevelTyped,
		LogJSON:         logJSONBool,
		sessRetention:   time.Duration(sessRetentionInt) * time.Hour,
		cleanupInterval: time.Duration(cleanupIntervalInt) * time.Hour,
		maxLogLines:     int(maxLogLinesInt),
	}, nil
}

func run(
	ctx context.Context,
	getENV func(key string) string,
) error {
	ctx, stop := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	params, err := NewRunParameters(
		getENV("DATA_PATH"),
		getENV("PORT"),
		getENV("LOG_INDENTED"),
		getENV("LOG_SOURCE"),
		getENV("LOG_LEVEL"),
		getENV("LOG_JSON"),
		getENV("CLEAN_UP_INTERVAL"),
		getENV("SESSION_RETENTION"),
		getENV("MAX_LOG_LINES"),
	)
	if err != nil {
		return fmt.Errorf("bad envirement variable: %w", err)
	}

	// App composing
	srv, err := wire.BuildServer(ctx, wire.Config{
		DataPath:  params.DataPath,
		Indented:  params.LogIndented,
		LogLevel:  params.LogLevel,
		AddSource: params.LogSource,
		JSON:      params.LogJSON,
	})
	if err != nil {
		return err
	}

	httpServer := http.Server{
		Addr:    fmt.Sprintf(":%s", params.Port),
		Handler: srv,
	}

	slog.Warn("start system")

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	// await
	<-ctx.Done()
	if err := httpServer.Shutdown(context.Background()); err != nil {
		return err
	}

	return nil
}

func main() {

	ctx := context.Background()
	if err := run(ctx, os.Getenv); err != nil {
		slog.Error("system run", "error", err)
		os.Exit(1)
	}
	slog.Warn("system shutdown")
}
