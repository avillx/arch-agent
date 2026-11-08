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
	logIndented := getBoolEnv("AGENT_LOG_INDENTED", false)
	logSource := getBoolEnv("AGENT_LOG_SOURCE", false)
	logLevel := getLoglevelEnv("AGENT_LOG_LEVEL", slog.LevelWarn)
	logJSON := getBoolEnv("AGENT_LOG_JSON", false)

	// ENV server config
	port := string(getInt64Env("AGENT_PORT", 8080))
	datapath := getEnv("AGENT_DATAPATH", "./agent-data")
	cleanupInterval := time.Duration(getInt64Env("AGENT_CLEAN_UP_INTERVAL", 12)) * time.Hour
	sessRetention := time.Duration(getInt64Env("AGENT_SESSION_RETENTION", 240)) * time.Hour
	maxLogLines := int(getInt64Env("AGENT_MAX_LOGLINES", 1000))
	agentShell := getEnv("AGENT_SHELL", "")
	agentShellAttribues := strings.Split(getEnv("AGENT_SHELL_ATTRIBUTES", ""), " ")

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
		datapath, maxLogLines, sessRetention, cleanupInterval, agentShell, agentShellAttribues, logger,
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

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func getInt64Env(key string, def int64) int64 {
	envString, ok := os.LookupEnv(key)
	if !ok {
		fmt.Printf("%s: is not specified. Applyed default %d", key, def)
		return def
	}

	v, err := strconv.ParseInt(envString, 10, 64)
	if err != nil {
		fmt.Printf("%s: %s cant be parsed as int value. Applyed default %d", key, v, def)
		return def
	}
	return v
}

func getBoolEnv(key string, def bool) bool {
	envString, ok := os.LookupEnv(key)
	if !ok {
		fmt.Printf("%s: is not specified. Applyed default %d", key, def)
		return def
	}

	v, err := strconv.ParseBool(envString)
	if err != nil {
		fmt.Printf("%s: %s cant be parsed as bool value. Applyed default %d", key, v, def)
		return def
	}
	return v
}

func getLoglevelEnv(key string, def slog.Level) slog.Level {
	envString, ok := os.LookupEnv(key)
	if !ok {
		fmt.Printf("%s: is not specified. Applyed default %s", key, def.String())
		return def
	}

	ll, err := logging.ToLogLevel(envString)
	if err != nil {
		fmt.Printf("%s: %s cant be parsed as log level value", key, def.String())
		return def
	}

	return ll
}
