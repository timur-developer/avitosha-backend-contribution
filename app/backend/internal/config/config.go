package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type LogLevel string

const (
	AppEnvDev  = "dev"
	AppEnvTest = "test"
	AppEnvProd = "prod"

	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"

	defaultAccessTokenTTL  = 15 * time.Minute
	defaultRefreshTokenTTL = 30 * 24 * time.Hour
)

type Config struct {
	AppEnv           string
	HTTPAddr         string
	DatabaseURL      string
	FrontendOrigin   string
	JWTSigningKey    string
	JWTIssuer        string
	JWTAudience      string
	LogLevel         LogLevel
	ShutdownTimeout  time.Duration
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
}

func Load() (Config, error) {
	return LoadFromEnv(os.Getenv)
}

func LoadFromEnv(getenv func(string) string) (Config, error) {
	cfg := Config{
		AppEnv:           envOrDefault(getenv, "APP_ENV", AppEnvDev),
		HTTPAddr:         strings.TrimSpace(getenv("HTTP_ADDR")),
		DatabaseURL:      strings.TrimSpace(getenv("DATABASE_URL")),
		FrontendOrigin:   strings.TrimSpace(getenv("FRONTEND_ORIGIN")),
		JWTSigningKey:    getenv("JWT_SIGNING_KEY"),
		JWTIssuer:        strings.TrimSpace(getenv("JWT_ISSUER")),
		JWTAudience:      strings.TrimSpace(getenv("JWT_AUDIENCE")),
		LogLevel:         LogLevel(envOrDefault(getenv, "LOG_LEVEL", string(LogLevelInfo))),
		ShutdownTimeout:  5 * time.Second,
		HTTPReadTimeout:  5 * time.Second,
		HTTPWriteTimeout: 10 * time.Second,
		HTTPIdleTimeout:  60 * time.Second,
		AccessTokenTTL:   defaultAccessTokenTTL,
		RefreshTokenTTL:  defaultRefreshTokenTTL,
	}

	if value := strings.TrimSpace(getenv("SHUTDOWN_TIMEOUT")); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT must be a duration: %w", err)
		}
		cfg.ShutdownTimeout = timeout
	}
	if value := strings.TrimSpace(getenv("ACCESS_TOKEN_TTL")); value != "" {
		ttl, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("ACCESS_TOKEN_TTL must be a duration: %w", err)
		}
		cfg.AccessTokenTTL = ttl
	}
	if value := strings.TrimSpace(getenv("REFRESH_TOKEN_TTL")); value != "" {
		ttl, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("REFRESH_TOKEN_TTL must be a duration: %w", err)
		}
		cfg.RefreshTokenTTL = ttl
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) Validate() error {
	if cfg.HTTPAddr == "" {
		return fmt.Errorf("HTTP_ADDR is required")
	}
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.FrontendOrigin == "" {
		return fmt.Errorf("FRONTEND_ORIGIN is required")
	}
	if cfg.JWTSigningKey == "" {
		return fmt.Errorf("JWT_SIGNING_KEY is required")
	}
	if cfg.JWTIssuer == "" {
		return fmt.Errorf("JWT_ISSUER is required")
	}
	if cfg.JWTAudience == "" {
		return fmt.Errorf("JWT_AUDIENCE is required")
	}

	switch cfg.AppEnv {
	case AppEnvDev, AppEnvTest, AppEnvProd:
	default:
		return fmt.Errorf("APP_ENV must be one of: dev, test, prod")
	}

	switch cfg.LogLevel {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error")
	}

	if cfg.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be positive")
	}
	if cfg.AccessTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be positive")
	}
	if cfg.RefreshTokenTTL <= 0 {
		return fmt.Errorf("REFRESH_TOKEN_TTL must be positive")
	}

	return nil
}

func (level LogLevel) Level() slog.Level {
	switch level {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func envOrDefault(getenv func(string) string, key, fallback string) string {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
