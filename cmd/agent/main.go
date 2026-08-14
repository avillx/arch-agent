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
)

type RunParameters struct {
	DataPath           string
	ConsolidationModel string
	ObserverModel      string
	Port               string
	LogPretty          string
	LogLevel           string
}

func NewRunParameters(
	dataPath string,
	consolidationModel string,
	observerModel string,
	port string,
	logPretty string,
	logLevel string,
) (RunParameters, error) {

	if dataPath == "" {
		dataPath = "."
	}

	if port == "" {
		port = "8080"
	}

	if logPretty == "" {
		logPretty = "false"
	}

	if logLevel == "" {
		logLevel = "error"
	}

	if consolidationModel == "" {
		return RunParameters{}, fmt.Errorf("consolidation model is not set")
	}

	if observerModel == "" {
		return RunParameters{}, fmt.Errorf("observer model is not set")
	}

	return RunParameters{
		DataPath:           dataPath,
		ConsolidationModel: consolidationModel,
		ObserverModel:      observerModel,
		Port:               port,
		LogPretty:          logPretty,
		LogLevel:           logLevel,
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
		getENV("CONSOLIDATION_MODEL"),
		getENV("OBSERVER_MODEL"),
		getENV("PORT"),
		getENV("LOG_PRETTY"),
		getENV("LOG_LEVEL"),
	)
	if err != nil {
		return fmt.Errorf("bad envirement variable: %w", err)
	}

	// logging
	logPrettyBool, err := strconv.ParseBool(params.LogPretty)
	if err != nil {
		return err
	}

	logLevel, err := logging.ToLogLevel(params.LogLevel)
	if err != nil {
		return err
	}

	logging.Set(logPrettyBool, logLevel)

	// App composing
	srv, err := wire.BuildServer(ctx, wire.Config{
		DataPath:           params.DataPath,
		ConsolidationModel: params.ConsolidationModel,
		ObserverModel:      params.ObserverModel,
	})
	if err != nil {
		return err
	}

	httpServer := http.Server{
		Addr:    fmt.Sprintf(":%s", params.Port),
		Handler: srv,
	}

	slog.Warn("start server")

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
		slog.Error("server run", "error", err)
		os.Exit(1)
	}

	slog.Warn("server shutdown")
}
