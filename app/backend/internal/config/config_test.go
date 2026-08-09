package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadFromEnv(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"APP_ENV":           "test",
		"HTTP_ADDR":         "127.0.0.1:8080",
		"DATABASE_URL":      "postgres://postgres:postgres@localhost:5432/avitosha?sslmode=disable",
		"FRONTEND_ORIGIN":   "http://localhost:3000",
		"JWT_SIGNING_KEY":   "test-signing-key",
		"JWT_ISSUER":        "avitosha",
		"JWT_AUDIENCE":      "avitosha-web",
		"LOG_LEVEL":         "debug",
		"SHUTDOWN_TIMEOUT":  "3s",
		"ACCESS_TOKEN_TTL":  "20m",
		"REFRESH_TOKEN_TTL": "720h",
	}

	cfg, err := LoadFromEnv(mapGetter(env))
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.AppEnv != AppEnvTest {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, AppEnvTest)
	}
	if cfg.HTTPAddr != "127.0.0.1:8080" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.DatabaseURL == "" {
		t.Fatal("DatabaseURL is empty")
	}
	if cfg.LogLevel != LogLevelDebug {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, LogLevelDebug)
	}
	if cfg.ShutdownTimeout != 3*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 3s", cfg.ShutdownTimeout)
	}
	if cfg.AccessTokenTTL != 20*time.Minute {
		t.Fatalf("AccessTokenTTL = %s, want 20m", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 30*24*time.Hour {
		t.Fatalf("RefreshTokenTTL = %s, want 720h", cfg.RefreshTokenTTL)
	}
}

func TestLoadFromEnvUsesDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromEnv(mapGetter(map[string]string{
		"HTTP_ADDR":       "127.0.0.1:8080",
		"DATABASE_URL":    "postgres://postgres:postgres@localhost:5432/avitosha?sslmode=disable",
		"FRONTEND_ORIGIN": "http://localhost:3000",
		"JWT_SIGNING_KEY": "test-signing-key",
		"JWT_ISSUER":      "avitosha",
		"JWT_AUDIENCE":    "avitosha-web",
	}))
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.AppEnv != AppEnvDev {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, AppEnvDev)
	}
	if cfg.LogLevel != LogLevelInfo {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, LogLevelInfo)
	}
	if cfg.ShutdownTimeout != 5*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 5s", cfg.ShutdownTimeout)
	}
	if cfg.AccessTokenTTL != defaultAccessTokenTTL {
		t.Fatalf("AccessTokenTTL = %s, want %s", cfg.AccessTokenTTL, defaultAccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != defaultRefreshTokenTTL {
		t.Fatalf("RefreshTokenTTL = %s, want %s", cfg.RefreshTokenTTL, defaultRefreshTokenTTL)
	}
}

func TestLoadFromEnvRequiresHTTPAddr(t *testing.T) {
	t.Parallel()

	_, err := LoadFromEnv(mapGetter(map[string]string{}))
	if err == nil {
		t.Fatal("LoadFromEnv() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "HTTP_ADDR") {
		t.Fatalf("error = %q, want HTTP_ADDR", err)
	}
}

func TestLoadFromEnvRejectsInvalidLogLevel(t *testing.T) {
	t.Parallel()

	_, err := LoadFromEnv(mapGetter(map[string]string{
		"HTTP_ADDR":       "127.0.0.1:8080",
		"DATABASE_URL":    "postgres://postgres:postgres@localhost:5432/avitosha?sslmode=disable",
		"FRONTEND_ORIGIN": "http://localhost:3000",
		"JWT_SIGNING_KEY": "test-signing-key",
		"JWT_ISSUER":      "avitosha",
		"JWT_AUDIENCE":    "avitosha-web",
		"LOG_LEVEL":       "verbose",
	}))
	if err == nil {
		t.Fatal("LoadFromEnv() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Fatalf("error = %q, want LOG_LEVEL", err)
	}
}

func TestLoadFromEnvRequiresDatabaseURL(t *testing.T) {
	t.Parallel()

	_, err := LoadFromEnv(mapGetter(map[string]string{
		"HTTP_ADDR":       "127.0.0.1:8080",
		"FRONTEND_ORIGIN": "http://localhost:3000",
		"JWT_SIGNING_KEY": "test-signing-key",
		"JWT_ISSUER":      "avitosha",
		"JWT_AUDIENCE":    "avitosha-web",
	}))
	if err == nil {
		t.Fatal("LoadFromEnv() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("error = %q, want DATABASE_URL", err)
	}
}

func TestLoadFromEnvRequiresFrontendOrigin(t *testing.T) {
	t.Parallel()

	_, err := LoadFromEnv(mapGetter(map[string]string{
		"HTTP_ADDR":       "127.0.0.1:8080",
		"DATABASE_URL":    "postgres://postgres:postgres@localhost:5432/avitosha?sslmode=disable",
		"JWT_SIGNING_KEY": "test-signing-key",
		"JWT_ISSUER":      "avitosha",
		"JWT_AUDIENCE":    "avitosha-web",
	}))
	if err == nil {
		t.Fatal("LoadFromEnv() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "FRONTEND_ORIGIN") {
		t.Fatalf("error = %q, want FRONTEND_ORIGIN", err)
	}
}

func TestLoadFromEnvRequiresJWTConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "missing signing key",
			env: map[string]string{
				"HTTP_ADDR":       "127.0.0.1:8080",
				"DATABASE_URL":    "postgres://postgres:postgres@localhost:5432/avitosha?sslmode=disable",
				"FRONTEND_ORIGIN": "http://localhost:3000",
				"JWT_ISSUER":      "avitosha",
				"JWT_AUDIENCE":    "avitosha-web",
			},
			want: "JWT_SIGNING_KEY",
		},
		{
			name: "missing issuer",
			env: map[string]string{
				"HTTP_ADDR":       "127.0.0.1:8080",
				"DATABASE_URL":    "postgres://postgres:postgres@localhost:5432/avitosha?sslmode=disable",
				"FRONTEND_ORIGIN": "http://localhost:3000",
				"JWT_SIGNING_KEY": "test-signing-key",
				"JWT_AUDIENCE":    "avitosha-web",
			},
			want: "JWT_ISSUER",
		},
		{
			name: "missing audience",
			env: map[string]string{
				"HTTP_ADDR":       "127.0.0.1:8080",
				"DATABASE_URL":    "postgres://postgres:postgres@localhost:5432/avitosha?sslmode=disable",
				"FRONTEND_ORIGIN": "http://localhost:3000",
				"JWT_SIGNING_KEY": "test-signing-key",
				"JWT_ISSUER":      "avitosha",
			},
			want: "JWT_AUDIENCE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := LoadFromEnv(mapGetter(tt.env))
			if err == nil {
				t.Fatal("LoadFromEnv() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %s", err, tt.want)
			}
		})
	}
}

func mapGetter(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
