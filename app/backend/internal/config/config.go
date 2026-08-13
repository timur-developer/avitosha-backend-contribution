package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
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
	defaultProxyAPIBaseURL = "https://api.proxyapi.ru/openrouter/v1"
	defaultProxyAPIModel   = "qwen/qwen-2.5-7b-instruct"
	defaultProxyAPITimeout = 4 * time.Second
	defaultS3Endpoint      = "/storage"
	defaultS3Region        = "us-east-1"
	defaultS3UploadTTL     = 10 * time.Minute
	defaultS3MaxFileSize   = int64(10 * 1024 * 1024)
)

type Config struct {
	AppEnv            string
	HTTPAddr          string
	DatabaseURL       string
	FrontendOrigin    string
	JWTSigningKey     string
	JWTIssuer         string
	JWTAudience       string
	LogLevel          LogLevel
	ShutdownTimeout   time.Duration
	HTTPReadTimeout   time.Duration
	HTTPWriteTimeout  time.Duration
	HTTPIdleTimeout   time.Duration
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	ProxyAPIKey       string
	ProxyAPIBaseURL   string
	ProxyAPIModel     string
	ProxyAPITimeout   time.Duration
	GRPCAddr          string
	AuthGRPCAddr      string
	GameGRPCAddr      string
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3PublicBaseURL   string
	S3UploadTTL       time.Duration
	S3MaxFileSize     int64
}

func Load() (Config, error) {
	return LoadFromEnv(os.Getenv)
}

func LoadGateway() (Config, error) {
	return loadForRole(os.Getenv, Config.ValidateGateway)
}

func LoadAuthService() (Config, error) {
	return loadForRole(os.Getenv, Config.ValidateAuthService)
}

func LoadGameService() (Config, error) {
	return loadForRole(os.Getenv, Config.ValidateGameService)
}

func LoadFromEnv(getenv func(string) string) (Config, error) {
	return loadForRole(getenv, Config.Validate)
}

func loadForRole(getenv func(string) string, validate func(Config) error) (Config, error) {
	cfg := Config{
		AppEnv:            envOrDefault(getenv, "APP_ENV", AppEnvDev),
		HTTPAddr:          strings.TrimSpace(getenv("HTTP_ADDR")),
		DatabaseURL:       strings.TrimSpace(getenv("DATABASE_URL")),
		FrontendOrigin:    strings.TrimSpace(getenv("FRONTEND_ORIGIN")),
		JWTSigningKey:     getenv("JWT_SIGNING_KEY"),
		JWTIssuer:         strings.TrimSpace(getenv("JWT_ISSUER")),
		JWTAudience:       strings.TrimSpace(getenv("JWT_AUDIENCE")),
		LogLevel:          LogLevel(envOrDefault(getenv, "LOG_LEVEL", string(LogLevelInfo))),
		ShutdownTimeout:   5 * time.Second,
		HTTPReadTimeout:   5 * time.Second,
		HTTPWriteTimeout:  10 * time.Second,
		HTTPIdleTimeout:   60 * time.Second,
		AccessTokenTTL:    defaultAccessTokenTTL,
		RefreshTokenTTL:   defaultRefreshTokenTTL,
		ProxyAPIKey:       strings.TrimSpace(getenv("PROXYAPI_API_KEY")),
		ProxyAPIBaseURL:   envOrDefault(getenv, "PROXYAPI_BASE_URL", defaultProxyAPIBaseURL),
		ProxyAPIModel:     envOrDefault(getenv, "PROXYAPI_MODEL", defaultProxyAPIModel),
		ProxyAPITimeout:   defaultProxyAPITimeout,
		GRPCAddr:          envOrDefault(getenv, "GRPC_ADDR", ":9090"),
		AuthGRPCAddr:      envOrDefault(getenv, "AUTH_GRPC_ADDR", "127.0.0.1:9091"),
		GameGRPCAddr:      envOrDefault(getenv, "GAME_GRPC_ADDR", "127.0.0.1:9092"),
		S3Endpoint:        envOrDefault(getenv, "S3_ENDPOINT", defaultS3Endpoint),
		S3Region:          envOrDefault(getenv, "S3_REGION", defaultS3Region),
		S3Bucket:          strings.TrimSpace(getenv("S3_BUCKET")),
		S3AccessKeyID:     strings.TrimSpace(getenv("S3_ACCESS_KEY_ID")),
		S3SecretAccessKey: strings.TrimSpace(getenv("S3_SECRET_ACCESS_KEY")),
		S3PublicBaseURL:   strings.TrimRight(strings.TrimSpace(getenv("S3_PUBLIC_BASE_URL")), "/"),
		S3UploadTTL:       defaultS3UploadTTL,
		S3MaxFileSize:     defaultS3MaxFileSize,
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
	if value := strings.TrimSpace(getenv("PROXYAPI_TIMEOUT")); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("PROXYAPI_TIMEOUT must be a duration: %w", err)
		}
		cfg.ProxyAPITimeout = timeout
	}
	if value := strings.TrimSpace(getenv("S3_UPLOAD_TTL")); value != "" {
		ttl, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("S3_UPLOAD_TTL must be a duration: %w", err)
		}
		cfg.S3UploadTTL = ttl
	}
	if value := strings.TrimSpace(getenv("S3_MAX_FILE_SIZE")); value != "" {
		size, err := parsePositiveInt64(value)
		if err != nil {
			return Config{}, fmt.Errorf("S3_MAX_FILE_SIZE must be bytes: %w", err)
		}
		cfg.S3MaxFileSize = size
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) ValidateGateway() error {
	if err := cfg.validateRuntime(); err != nil {
		return err
	}
	if cfg.HTTPAddr == "" {
		return fmt.Errorf("HTTP_ADDR is required")
	}
	if cfg.FrontendOrigin == "" {
		return fmt.Errorf("FRONTEND_ORIGIN is required")
	}
	if cfg.RefreshTokenTTL <= 0 {
		return fmt.Errorf("REFRESH_TOKEN_TTL must be positive")
	}
	if cfg.AuthGRPCAddr == "" || cfg.GameGRPCAddr == "" {
		return fmt.Errorf("AUTH_GRPC_ADDR and GAME_GRPC_ADDR must not be empty")
	}
	if err := cfg.validateObjectStorage(); err != nil {
		return err
	}
	return nil
}

func (cfg Config) ObjectStorageEnabled() bool { return cfg.S3Bucket != "" }

func (cfg Config) validateObjectStorage() error {
	configured := cfg.S3Bucket != "" || cfg.S3AccessKeyID != "" || cfg.S3SecretAccessKey != "" || cfg.S3PublicBaseURL != ""
	if !configured {
		return nil
	}
	if cfg.S3Bucket == "" || cfg.S3AccessKeyID == "" || cfg.S3SecretAccessKey == "" {
		return fmt.Errorf("S3_BUCKET, S3_ACCESS_KEY_ID and S3_SECRET_ACCESS_KEY must be set together")
	}
	if cfg.S3Endpoint == "" || cfg.S3Region == "" || cfg.S3UploadTTL <= 0 || cfg.S3MaxFileSize <= 0 {
		return fmt.Errorf("S3 endpoint/region must not be empty and upload limits must be positive")
	}
	return nil
}

func parsePositiveInt64(value string) (int64, error) {
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil || result <= 0 {
		return 0, fmt.Errorf("value must be a positive integer")
	}
	return result, nil
}

func (cfg Config) ValidateAuthService() error {
	if err := cfg.validateRuntime(); err != nil {
		return err
	}
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
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
	if cfg.AccessTokenTTL <= 0 || cfg.RefreshTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL and REFRESH_TOKEN_TTL must be positive")
	}
	if cfg.GRPCAddr == "" {
		return fmt.Errorf("GRPC_ADDR must not be empty")
	}
	return nil
}

func (cfg Config) ValidateGameService() error {
	if err := cfg.validateRuntime(); err != nil {
		return err
	}
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.GRPCAddr == "" {
		return fmt.Errorf("GRPC_ADDR must not be empty")
	}
	if cfg.ProxyAPIBaseURL == "" || cfg.ProxyAPIModel == "" || cfg.ProxyAPITimeout <= 0 {
		return fmt.Errorf("ProxyAPI URL/model must not be empty and timeout must be positive")
	}
	return nil
}

func (cfg Config) validateRuntime() error {
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
	return nil
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
	if cfg.ProxyAPIBaseURL == "" {
		return fmt.Errorf("PROXYAPI_BASE_URL must not be empty")
	}
	if cfg.ProxyAPIModel == "" {
		return fmt.Errorf("PROXYAPI_MODEL must not be empty")
	}
	if cfg.ProxyAPITimeout <= 0 {
		return fmt.Errorf("PROXYAPI_TIMEOUT must be positive")
	}
	if cfg.GRPCAddr == "" || cfg.AuthGRPCAddr == "" || cfg.GameGRPCAddr == "" {
		return fmt.Errorf("GRPC_ADDR, AUTH_GRPC_ADDR and GAME_GRPC_ADDR must not be empty")
	}
	if err := cfg.validateObjectStorage(); err != nil {
		return err
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
