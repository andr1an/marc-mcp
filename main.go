package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andr1an/marc-mcp/internal/config"
	"github.com/andr1an/marc-mcp/internal/httpserver"
	"github.com/andr1an/marc-mcp/internal/tools"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("version=%s commit=%s date=%s\n", version, commit, date)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	srv, err := httpserver.New(cfg, logger, version)
	if err != nil {
		logger.Error("failed to build server", "error", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("starting server",
			"addr", srv.Addr,
			"version", version,
			"commit", commit,
			"date", date,
			"read_timeout", cfg.ReadTimeout,
			"write_timeout", cfg.WriteTimeout,
			"idle_timeout", cfg.IdleTimeout,
		)

		if err := srv.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
			logger.Error("server exited with error", "error", err)
			os.Exit(1)
		}
	}()

	waitForShutdown(logger, srv, cfg.ShutdownTimeout)
}

func newLogger(level string) *slog.Logger {
	var slogLevel slog.Level

	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel})
	return slog.New(handler)
}

func waitForShutdown(logger *slog.Logger, srv *httpserver.Server, timeout time.Duration) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}

	if err := tools.Close(); err != nil {
		logger.Error("failed to close marc client", "error", err)
	}

	logger.Info("server stopped")
}
