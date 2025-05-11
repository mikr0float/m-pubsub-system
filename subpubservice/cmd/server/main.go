package main

import (
	"log/slog"
	"os"

	"github.com/mikr0float/m-pubsub-system/subpubservice/config"
	"github.com/mikr0float/m-pubsub-system/subpubservice/internal/server"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	srv := server.New(cfg, logger)
	if err := srv.Start(); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	srv.WaitForShutdown()
}
