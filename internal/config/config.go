package config

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/svetlyopet/heimdallr/internal/constants"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/logger"
)

const (
	bootstrapRootPasswordEnv = "HEIMDALLR_BOOTSTRAP_ROOT_PASSWORD"
	loginRateLimitMaxEnv     = "HEIMDALLR_LOGIN_RATE_LIMIT_MAX"
	loginRateLimitWindowEnv  = "HEIMDALLR_LOGIN_RATE_LIMIT_WINDOW"
	sessionTokenTTLEnv       = "HEIMDALLR_SESSION_TOKEN_TTL"
	apiTokenDefaultTTLEnv    = "HEIMDALLR_API_TOKEN_DEFAULT_TTL"
	apiTokenMaxTTLEnv        = "HEIMDALLR_API_TOKEN_MAX_TTL"
	sessionCookieNameEnv     = "HEIMDALLR_SESSION_COOKIE_NAME"
	csrfCookieNameEnv        = "HEIMDALLR_CSRF_COOKIE_NAME"
	cookieSecureEnv          = "HEIMDALLR_COOKIE_SECURE"
	readHeaderTimeoutEnv     = "HEIMDALLR_READ_HEADER_TIMEOUT"
	readTimeoutEnv           = "HEIMDALLR_READ_TIMEOUT"
	writeTimeoutEnv          = "HEIMDALLR_WRITE_TIMEOUT"
	idleTimeoutEnv           = "HEIMDALLR_IDLE_TIMEOUT"
	maxHeaderBytesEnv        = "HEIMDALLR_MAX_HEADER_BYTES"
	maxRequestBodyBytesEnv   = "HEIMDALLR_MAX_REQUEST_BODY_BYTES"
	maxDecodedOutputBytesEnv = "HEIMDALLR_MAX_DECODED_OUTPUT_BYTES"
	maxPaginationLimitEnv    = "HEIMDALLR_MAX_PAGINATION_LIMIT"
	hardMaximumAPITokenTTL   = 365 * 24 * time.Hour
)

type ServerConfig struct {
	Host                  string
	Port                  string
	ReadHeaderTimeout     time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	IdleTimeout           time.Duration
	MaxHeaderBytes        int
	MaxRequestBodyBytes   int64
	MaxDecodedOutputBytes int64
	MaxPaginationLimit    int
}

type AuthConfig struct {
	BootstrapRootPassword string
	LoginRateLimitMax     int
	LoginRateLimitWindow  time.Duration
	SessionTokenTTL       time.Duration
	DefaultAPITokenTTL    time.Duration
	MaxAPITokenTTL        time.Duration
	SessionCookieName     string
	CSRFCookieName        string
	CookieSecure          bool
}

type AppConfig struct {
	Server   ServerConfig
	Database database.Config
	Logger   logger.Config
	Auth     AuthConfig
}

func LoadFromFlags(args []string, env func(string) string) (AppConfig, error) {
	if env == nil {
		env = os.Getenv
	}

	fs := flag.NewFlagSet("heimdallr", flag.ContinueOnError)
	logFormat := fs.String("log-format", "text", "log format: text or json")
	logLevel := fs.String("log-level", "info", "log level: debug, info, warn, or error")
	serverName := fs.String("server-name", constants.ApiDefaultHost, "server name")
	serverPort := fs.String("server-port", constants.ApiDefaultPort, "server port")
	databasePath := fs.String("database-path", constants.AppDefaultName+".db", "sqlite database path when DATABASE_URL is unset")

	if err := fs.Parse(args); err != nil {
		return AppConfig{}, fmt.Errorf("parse flags: %w", err)
	}

	releaseMode := isReleaseMode(env)
	cookieSecure, err := parseBoolEnv(env, cookieSecureEnv, releaseMode)
	if err != nil {
		return AppConfig{}, err
	}
	if releaseMode && !cookieSecure {
		return AppConfig{}, fmt.Errorf("%s must be true in release/production mode", cookieSecureEnv)
	}

	defaultAPITokenTTL := parseDurationEnv(env, apiTokenDefaultTTLEnv, 90*24*time.Hour)
	maxAPITokenTTL := parseDurationEnv(env, apiTokenMaxTTLEnv, hardMaximumAPITokenTTL)
	if maxAPITokenTTL > hardMaximumAPITokenTTL {
		return AppConfig{}, fmt.Errorf("%s must not exceed %s", apiTokenMaxTTLEnv, hardMaximumAPITokenTTL)
	}
	if defaultAPITokenTTL > maxAPITokenTTL {
		return AppConfig{}, fmt.Errorf("%s must not exceed %s", apiTokenDefaultTTLEnv, apiTokenMaxTTLEnv)
	}
	maxPaginationLimit := parseIntEnv(env, maxPaginationLimitEnv, 100)
	if maxPaginationLimit > 100 {
		return AppConfig{}, fmt.Errorf("%s must not exceed 100", maxPaginationLimitEnv)
	}

	return AppConfig{
		Server: ServerConfig{
			Host:                  *serverName,
			Port:                  *serverPort,
			ReadHeaderTimeout:     parseDurationEnv(env, readHeaderTimeoutEnv, 5*time.Second),
			ReadTimeout:           parseDurationEnv(env, readTimeoutEnv, 15*time.Second),
			WriteTimeout:          parseDurationEnv(env, writeTimeoutEnv, 30*time.Second),
			IdleTimeout:           parseDurationEnv(env, idleTimeoutEnv, 60*time.Second),
			MaxHeaderBytes:        parseIntEnv(env, maxHeaderBytesEnv, 1<<20),
			MaxRequestBodyBytes:   int64(parseIntEnv(env, maxRequestBodyBytesEnv, 5<<20)),
			MaxDecodedOutputBytes: int64(parseIntEnv(env, maxDecodedOutputBytesEnv, 4<<20)),
			MaxPaginationLimit:    maxPaginationLimit,
		},
		Database: database.Config{
			DatabaseURL:  strings.TrimSpace(env("DATABASE_URL")),
			DatabasePath: *databasePath,
		},
		Logger: logger.Config{
			Format: logger.Format(*logFormat),
			Level:  parseLogLevel(*logLevel),
			Output: os.Stdout,
		},
		Auth: AuthConfig{
			BootstrapRootPassword: strings.TrimSpace(env(bootstrapRootPasswordEnv)),
			LoginRateLimitMax:     parseIntEnv(env, loginRateLimitMaxEnv, 10),
			LoginRateLimitWindow:  parseDurationEnv(env, loginRateLimitWindowEnv, 15*time.Minute),
			SessionTokenTTL:       parseDurationEnv(env, sessionTokenTTLEnv, 24*time.Hour),
			DefaultAPITokenTTL:    defaultAPITokenTTL,
			MaxAPITokenTTL:        maxAPITokenTTL,
			SessionCookieName:     envOrDefault(env, sessionCookieNameEnv, "heimdallr_session"),
			CSRFCookieName:        envOrDefault(env, csrfCookieNameEnv, "heimdallr_csrf"),
			CookieSecure:          cookieSecure,
		},
	}, nil
}

func DefaultTestConfig(output io.Writer) AppConfig {
	if output == nil {
		output = io.Discard
	}

	return AppConfig{
		Server: ServerConfig{
			Host:                  constants.ApiDefaultHost,
			Port:                  constants.ApiDefaultPort,
			ReadHeaderTimeout:     5 * time.Second,
			ReadTimeout:           15 * time.Second,
			WriteTimeout:          30 * time.Second,
			IdleTimeout:           60 * time.Second,
			MaxHeaderBytes:        1 << 20,
			MaxRequestBodyBytes:   5 << 20,
			MaxDecodedOutputBytes: 4 << 20,
			MaxPaginationLimit:    100,
		},
		Database: database.Config{
			DatabasePath: ":memory:",
		},
		Logger: logger.Config{
			Format: logger.FormatText,
			Level:  slog.LevelError,
			Output: output,
		},
		Auth: AuthConfig{
			BootstrapRootPassword: "IntegrationTestPassword12!",
			LoginRateLimitMax:     100,
			LoginRateLimitWindow:  time.Minute,
			SessionTokenTTL:       24 * time.Hour,
			DefaultAPITokenTTL:    90 * 24 * time.Hour,
			MaxAPITokenTTL:        365 * 24 * time.Hour,
			SessionCookieName:     "heimdallr_session",
			CSRFCookieName:        "heimdallr_csrf",
			CookieSecure:          false,
		},
	}
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func parseIntEnv(env func(string) string, key string, fallback int) int {
	value := strings.TrimSpace(env(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func parseDurationEnv(env func(string) string, key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(env(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func parseBoolEnv(env func(string) string, key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(env(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

func envOrDefault(env func(string) string, key string, fallback string) string {
	value := strings.TrimSpace(env(key))
	if value == "" {
		return fallback
	}

	return value
}

func isReleaseMode(env func(string) string) bool {
	ginMode := strings.ToLower(strings.TrimSpace(env("GIN_MODE")))
	appEnvironment := strings.ToLower(strings.TrimSpace(env("HEIMDALLR_ENV")))
	return ginMode == "release" || appEnvironment == "production"
}
