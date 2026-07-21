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

func run(
	ctx context.Context,
	getENV func(key string) (string, bool),
) error {
	ctx, stop := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	dataPath, ok := getENV("DATA_PATH")
	if !ok {
		dataPath = "."
	}

	consolidationModel, ok := getENV("CONSOLIDATION_MODEL")
	if !ok {
		return fmt.Errorf("env 'CONSOLIDATION_MODEL' must be non nil")
	}

	observerModel, ok := getENV("OBSERVER_MODEL")
	if !ok {
		return fmt.Errorf("env 'OBSERVER_MODEL' must be non nil")
	}

	port, ok := getENV("PORT")
	if !ok {
		// default
		port = "8080"
	}

	publicURL, ok := getENV("PUBLIC_URL")
	if !ok {
		// default
		publicURL = fmt.Sprintf("localhost:%s", port)
	}

	logPretty, ok := getENV("LOG_PRETTY")
	if !ok {
		logPretty = "false"
	}

	logLevel, ok := getENV("LOG_LEVEL")
	if !ok {
		// default
		logLevel = "4"
	}

	// logging
	logPrettyBool, err := strconv.ParseBool(logPretty)
	if err != nil {
		return err
	}

	logLevelInt, err := strconv.Atoi(logLevel)
	if err != nil {
		return err
	}

	logging.Set(logPrettyBool, slog.Level(logLevelInt))

	// App composing
	srv, err := wire.BuildServer(ctx, wire.Config{
		PubURL:             publicURL,
		DataPath:           dataPath,
		ConsolidationModel: consolidationModel,
		ObserverModel:      observerModel,
	})
	if err != nil {
		return err
	}

	httpServer := http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: srv,
	}

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
	if err := run(ctx, os.LookupEnv); err != nil {
		slog.Error("server run", "error", err)
		os.Exit(1)
	}

	slog.Warn("server shutdown")
}
