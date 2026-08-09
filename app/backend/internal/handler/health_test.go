package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDatabasePinger struct {
	err error
}

func (f fakeDatabasePinger) Ping(_ context.Context) error {
	return f.err
}

func TestHealthEndpoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		dbErr      error
		wantStatus int
		wantBody   string
	}{
		{name: "live", path: "/health/live", wantStatus: http.StatusOK, wantBody: "{\"status\":\"ok\"}\n"},
		{name: "ready ok", path: "/health/ready", wantStatus: http.StatusOK, wantBody: "{\"status\":\"ok\"}\n"},
		{name: "ready unavailable", path: "/health/ready", dbErr: errors.New("dial tcp 127.0.0.1:5432: connection refused"), wantStatus: http.StatusServiceUnavailable, wantBody: "{\"status\":\"unavailable\"}\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newTestRouter(t, testRouterConfig{dbErr: tt.dbErr})
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if contentType := rec.Header().Get("Content-Type"); contentType != jsonContentType {
				t.Fatalf("Content-Type = %q, want %s", contentType, jsonContentType)
			}
			if body := rec.Body.String(); body != tt.wantBody {
				t.Fatalf("body = %q", body)
			}
		})
	}
}
