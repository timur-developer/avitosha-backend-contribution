package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/app"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/config"
)

func main() {
	cfg, err := config.LoadGateway()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "api failed: load config: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel.Level(),
	}))

	application, err := app.NewGateway(cfg, logger)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "api failed: %v\n", err)
		os.Exit(1)
	}

	if err := application.Run(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "api failed: %v\n", err)
		os.Exit(1)
	}
}
