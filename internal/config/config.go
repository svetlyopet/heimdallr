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
)

type ServerConfig struct {
	Host string
	Port string
}

type AuthConfig struct {
	BootstrapRootPassword string
	LoginRateLimitMax     int
	LoginRateLimitWindow  time.Duration
	SessionTokenTTL       time.Duration
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

	return AppConfig{
		Server: ServerConfig{
			Host: *serverName,
			Port: *serverPort,
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
		},
	}, nil
}

func DefaultTestConfig(output io.Writer) AppConfig {
	if output == nil {
		output = io.Discard
	}

	return AppConfig{
		Server: ServerConfig{
			Host: constants.ApiDefaultHost,
			Port: constants.ApiDefaultPort,
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
