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
	cfg, err := config.LoadAuthService()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "auth-service failed: load config: %v\n", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel.Level()}))
	application, err := app.NewAuthGRPCService(context.Background(), cfg, logger)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "auth-service failed: %v\n", err)
		os.Exit(1)
	}
	if err := application.Run(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "auth-service failed: %v\n", err)
		os.Exit(1)
	}
}
