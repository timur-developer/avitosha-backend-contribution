package app

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestAppRunStopsWhenContextIsCanceled(t *testing.T) {
	app := newApp(slog.Default(), &http.Server{
		Addr: ":0",
	}, nil, 2*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- app.Run(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("App.Run() did not stop after context cancellation")
	}
}
