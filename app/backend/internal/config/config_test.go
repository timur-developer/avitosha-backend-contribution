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
		"PROXYAPI_API_KEY":  "proxy-key",
		"PROXYAPI_BASE_URL": "https://proxy.example/openrouter/v1",
		"PROXYAPI_MODEL":    "qwen/test-model",
		"PROXYAPI_TIMEOUT":  "2s",
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
	if cfg.ProxyAPIKey != "proxy-key" || cfg.ProxyAPIBaseURL != "https://proxy.example/openrouter/v1" {
		t.Fatalf("ProxyAPI config = key %q, url %q", cfg.ProxyAPIKey, cfg.ProxyAPIBaseURL)
	}
	if cfg.ProxyAPIModel != "qwen/test-model" || cfg.ProxyAPITimeout != 2*time.Second {
		t.Fatalf("ProxyAPI model = %q, timeout = %s", cfg.ProxyAPIModel, cfg.ProxyAPITimeout)
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
	if cfg.ProxyAPIBaseURL != defaultProxyAPIBaseURL || cfg.ProxyAPIModel != defaultProxyAPIModel {
		t.Fatalf("ProxyAPI defaults = url %q, model %q", cfg.ProxyAPIBaseURL, cfg.ProxyAPIModel)
	}
	if cfg.ProxyAPITimeout != defaultProxyAPITimeout {
		t.Fatalf("ProxyAPITimeout = %s, want %s", cfg.ProxyAPITimeout, defaultProxyAPITimeout)
	}
}

func TestLoadFromEnvLoadsObjectStorage(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromEnv(mapGetter(map[string]string{
		"HTTP_ADDR": "127.0.0.1:8080", "DATABASE_URL": "postgres://test",
		"FRONTEND_ORIGIN": "http://localhost:3000", "JWT_SIGNING_KEY": "test-key",
		"JWT_ISSUER": "avitosha", "JWT_AUDIENCE": "avitosha-web",
		"S3_BUCKET": "photos", "S3_ACCESS_KEY_ID": "access", "S3_SECRET_ACCESS_KEY": "secret",
		"S3_PUBLIC_BASE_URL": "https://cdn.example.test/photos/", "S3_UPLOAD_TTL": "5m",
		"S3_MAX_FILE_SIZE": "2048",
	}))
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if !cfg.ObjectStorageEnabled() || cfg.S3Bucket != "photos" || cfg.S3UploadTTL != 5*time.Minute || cfg.S3MaxFileSize != 2048 {
		t.Fatalf("object storage config = %+v", cfg)
	}
	if cfg.S3PublicBaseURL != "https://cdn.example.test/photos" {
		t.Fatalf("S3PublicBaseURL = %q", cfg.S3PublicBaseURL)
	}
}

func TestLoadFromEnvRejectsPartialObjectStorageConfig(t *testing.T) {
	t.Parallel()

	_, err := LoadFromEnv(mapGetter(map[string]string{
		"HTTP_ADDR": "127.0.0.1:8080", "DATABASE_URL": "postgres://test",
		"FRONTEND_ORIGIN": "http://localhost:3000", "JWT_SIGNING_KEY": "test-key",
		"JWT_ISSUER": "avitosha", "JWT_AUDIENCE": "avitosha-web", "S3_BUCKET": "photos",
	}))
	if err == nil || !strings.Contains(err.Error(), "S3_ACCESS_KEY_ID") {
		t.Fatalf("error = %v, want incomplete S3 config error", err)
	}
}

func TestLoadFromEnvRejectsInvalidProxyAPITimeout(t *testing.T) {
	t.Parallel()
	_, err := LoadFromEnv(mapGetter(map[string]string{
		"HTTP_ADDR": "127.0.0.1:8080", "DATABASE_URL": "postgres://test",
		"FRONTEND_ORIGIN": "http://localhost:3000", "JWT_SIGNING_KEY": "test-key",
		"JWT_ISSUER": "avitosha", "JWT_AUDIENCE": "avitosha-web", "PROXYAPI_TIMEOUT": "0s",
	}))
	if err == nil || !strings.Contains(err.Error(), "PROXYAPI_TIMEOUT") {
		t.Fatalf("error = %v, want PROXYAPI_TIMEOUT", err)
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

func TestRoleValidationKeepsSecretsScopedToOwningService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      map[string]string
		validate func(Config) error
	}{
		{
			name: "gateway does not need database or JWT",
			env: map[string]string{
				"HTTP_ADDR": ":8080", "FRONTEND_ORIGIN": "http://localhost:3000",
				"AUTH_GRPC_ADDR": "auth:9091", "GAME_GRPC_ADDR": "game:9092",
			},
			validate: Config.ValidateGateway,
		},
		{
			name: "auth does not need HTTP or ProxyAPI key",
			env: map[string]string{
				"DATABASE_URL": "postgres://auth", "GRPC_ADDR": ":9091",
				"JWT_SIGNING_KEY": "secret", "JWT_ISSUER": "avitosha", "JWT_AUDIENCE": "web",
			},
			validate: Config.ValidateAuthService,
		},
		{
			name: "game does not need HTTP or JWT",
			env: map[string]string{
				"DATABASE_URL": "postgres://game", "GRPC_ADDR": ":9092",
			},
			validate: Config.ValidateGameService,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := loadForRole(mapGetter(test.env), test.validate); err != nil {
				t.Fatalf("loadForRole() error = %v", err)
			}
		})
	}
}

func mapGetter(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
