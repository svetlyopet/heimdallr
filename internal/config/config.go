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
	loginRateLimitMaxKeysEnv = "HEIMDALLR_LOGIN_RATE_LIMIT_MAX_KEYS"
	apiRateLimitIPRateEnv    = "HEIMDALLR_API_RATE_LIMIT_IP_RATE"
	apiRateLimitIPBurstEnv   = "HEIMDALLR_API_RATE_LIMIT_IP_BURST"
	apiRateLimitUserRateEnv  = "HEIMDALLR_API_RATE_LIMIT_USER_RATE"
	apiRateLimitUserBurstEnv = "HEIMDALLR_API_RATE_LIMIT_USER_BURST"
	apiRateLimitMaxKeysEnv   = "HEIMDALLR_API_RATE_LIMIT_MAX_KEYS"
	apiRateLimitStaleTTLEnv  = "HEIMDALLR_API_RATE_LIMIT_STALE_TTL"
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
	LoginRateLimitMaxKeys int
	APIRateLimitIPRate    float64
	APIRateLimitIPBurst   float64
	APIRateLimitUserRate  float64
	APIRateLimitUserBurst float64
	APIRateLimitMaxKeys   int
	APIRateLimitStaleTTL  time.Duration
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
	configPath := fs.String("config", "", "path to YAML config file")
	logFormat := fs.String("log-format", "text", "log format: text or json")
	logLevel := fs.String("log-level", "info", "log level: debug, info, warn, or error")
	serverName := fs.String("server-name", constants.ApiDefaultHost, "server name")
	serverPort := fs.String("server-port", constants.ApiDefaultPort, "server port")

	if err := fs.Parse(args); err != nil {
		return AppConfig{}, fmt.Errorf("parse flags: %w", err)
	}

	explicitFlags := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		explicitFlags[f.Name] = true
	})

	var fileCfg fileConfig
	if strings.TrimSpace(*configPath) != "" {
		var err error
		fileCfg, err = loadFileConfig(strings.TrimSpace(*configPath))
		if err != nil {
			return AppConfig{}, err
		}
	}

	releaseMode := isReleaseMode(env)
	cookieSecureFallback := releaseMode
	if fileCfg.Auth.CookieSecure != nil {
		cookieSecureFallback = *fileCfg.Auth.CookieSecure
	}
	cookieSecure, err := mergeBool(
		cookieSecureFallback,
		env,
		cookieSecureEnv,
		fileCfg.Auth.CookieSecure,
	)
	if err != nil {
		return AppConfig{}, err
	}
	if releaseMode && !cookieSecure {
		return AppConfig{}, fmt.Errorf("%s must be true in release/production mode", cookieSecureEnv)
	}

	defaultAPITokenTTL := mergeDuration(
		90*24*time.Hour,
		env,
		apiTokenDefaultTTLEnv,
		fileCfg.Auth.DefaultAPITokenTTL,
	)
	maxAPITokenTTL := mergeDuration(
		hardMaximumAPITokenTTL,
		env,
		apiTokenMaxTTLEnv,
		fileCfg.Auth.MaxAPITokenTTL,
	)
	if maxAPITokenTTL > hardMaximumAPITokenTTL {
		return AppConfig{}, fmt.Errorf("%s must not exceed %s", apiTokenMaxTTLEnv, hardMaximumAPITokenTTL)
	}
	if defaultAPITokenTTL > maxAPITokenTTL {
		return AppConfig{}, fmt.Errorf("%s must not exceed %s", apiTokenDefaultTTLEnv, apiTokenMaxTTLEnv)
	}

	maxPaginationLimit := mergeInt(
		100,
		env,
		maxPaginationLimitEnv,
		fileCfg.Server.MaxPaginationLimit,
	)
	if maxPaginationLimit > 100 {
		return AppConfig{}, fmt.Errorf("%s must not exceed 100", maxPaginationLimitEnv)
	}

	logFormatValue := mergeString(
		"text",
		fileCfg.Logger.Format,
		"",
		explicitFlags["log-format"],
		*logFormat,
	)
	logLevelValue := mergeString(
		"info",
		fileCfg.Logger.Level,
		"",
		explicitFlags["log-level"],
		*logLevel,
	)
	serverHost := mergeString(
		constants.ApiDefaultHost,
		fileCfg.Server.Host,
		"",
		explicitFlags["server-name"],
		*serverName,
	)
	serverPortValue := mergeString(
		constants.ApiDefaultPort,
		fileCfg.Server.Port,
		"",
		explicitFlags["server-port"],
		*serverPort,
	)

	return AppConfig{
		Server: ServerConfig{
			Host: serverHost,
			Port: serverPortValue,
			ReadHeaderTimeout: mergeDuration(
				5*time.Second,
				env,
				readHeaderTimeoutEnv,
				fileCfg.Server.ReadHeaderTimeout,
			),
			ReadTimeout: mergeDuration(
				15*time.Second,
				env,
				readTimeoutEnv,
				fileCfg.Server.ReadTimeout,
			),
			WriteTimeout: mergeDuration(
				30*time.Second,
				env,
				writeTimeoutEnv,
				fileCfg.Server.WriteTimeout,
			),
			IdleTimeout: mergeDuration(
				60*time.Second,
				env,
				idleTimeoutEnv,
				fileCfg.Server.IdleTimeout,
			),
			MaxHeaderBytes: mergeInt(
				1<<20,
				env,
				maxHeaderBytesEnv,
				fileCfg.Server.MaxHeaderBytes,
			),
			MaxRequestBodyBytes: int64(mergeInt(
				5<<20,
				env,
				maxRequestBodyBytesEnv,
				int(fileCfg.Server.MaxRequestBodyBytes),
			)),
			MaxDecodedOutputBytes: int64(mergeInt(
				4<<20,
				env,
				maxDecodedOutputBytesEnv,
				int(fileCfg.Server.MaxDecodedOutputBytes),
			)),
			MaxPaginationLimit: maxPaginationLimit,
		},
		Database: database.Config{
			DatabaseURL: mergeString("", fileCfg.Database.URL, strings.TrimSpace(env("DATABASE_URL")), false, ""),
		},
		Logger: logger.Config{
			Format: logger.Format(logFormatValue),
			Level:  parseLogLevel(logLevelValue),
			Output: os.Stdout,
		},
		Auth: AuthConfig{
			BootstrapRootPassword: mergeString(
				"",
				fileCfg.Auth.BootstrapRootPassword,
				strings.TrimSpace(env(bootstrapRootPasswordEnv)),
				false,
				"",
			),
			LoginRateLimitMax: mergeInt(
				10,
				env,
				loginRateLimitMaxEnv,
				fileCfg.Auth.LoginRateLimitMax,
			),
			LoginRateLimitWindow: mergeDuration(
				15*time.Minute,
				env,
				loginRateLimitWindowEnv,
				fileCfg.Auth.LoginRateLimitWindow,
			),
			LoginRateLimitMaxKeys: mergeInt(
				10_000,
				env,
				loginRateLimitMaxKeysEnv,
				fileCfg.Auth.LoginRateLimitMaxKeys,
			),
			APIRateLimitIPRate: mergeFloat(
				20,
				env,
				apiRateLimitIPRateEnv,
				fileCfg.Auth.APIRateLimitIPRate,
			),
			APIRateLimitIPBurst: mergeFloat(
				40,
				env,
				apiRateLimitIPBurstEnv,
				fileCfg.Auth.APIRateLimitIPBurst,
			),
			APIRateLimitUserRate: mergeFloat(
				50,
				env,
				apiRateLimitUserRateEnv,
				fileCfg.Auth.APIRateLimitUserRate,
			),
			APIRateLimitUserBurst: mergeFloat(
				100,
				env,
				apiRateLimitUserBurstEnv,
				fileCfg.Auth.APIRateLimitUserBurst,
			),
			APIRateLimitMaxKeys: mergeInt(
				10_000,
				env,
				apiRateLimitMaxKeysEnv,
				fileCfg.Auth.APIRateLimitMaxKeys,
			),
			APIRateLimitStaleTTL: mergeDuration(
				time.Hour,
				env,
				apiRateLimitStaleTTLEnv,
				fileCfg.Auth.APIRateLimitStaleTTL,
			),
			SessionTokenTTL: mergeDuration(
				24*time.Hour,
				env,
				sessionTokenTTLEnv,
				fileCfg.Auth.SessionTokenTTL,
			),
			DefaultAPITokenTTL: defaultAPITokenTTL,
			MaxAPITokenTTL:     maxAPITokenTTL,
			SessionCookieName: mergeString(
				"heimdallr_session",
				fileCfg.Auth.SessionCookieName,
				strings.TrimSpace(env(sessionCookieNameEnv)),
				false,
				"",
			),
			CSRFCookieName: mergeString(
				"heimdallr_csrf",
				fileCfg.Auth.CSRFCookieName,
				strings.TrimSpace(env(csrfCookieNameEnv)),
				false,
				"",
			),
			CookieSecure: cookieSecure,
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
			DatabaseURL: strings.TrimSpace(os.Getenv("TEST_POSTGRES_URL")),
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
			LoginRateLimitMaxKeys: 10_000,
			APIRateLimitIPRate:    20,
			APIRateLimitIPBurst:   40,
			APIRateLimitUserRate:  50,
			APIRateLimitUserBurst: 100,
			APIRateLimitMaxKeys:   10_000,
			APIRateLimitStaleTTL:  time.Hour,
			SessionTokenTTL:       24 * time.Hour,
			DefaultAPITokenTTL:    90 * 24 * time.Hour,
			MaxAPITokenTTL:        365 * 24 * time.Hour,
			SessionCookieName:     "heimdallr_session",
			CSRFCookieName:        "heimdallr_csrf",
			CookieSecure:          false,
		},
	}
}

func mergeString(defaultValue, fileValue, envValue string, flagSet bool, flagValue string) string {
	value := defaultValue
	if strings.TrimSpace(fileValue) != "" {
		value = strings.TrimSpace(fileValue)
	}
	if strings.TrimSpace(envValue) != "" {
		value = strings.TrimSpace(envValue)
	}
	if flagSet && strings.TrimSpace(flagValue) != "" {
		value = strings.TrimSpace(flagValue)
	}
	return value
}

func mergeInt(defaultValue int, env func(string) string, envKey string, fileValue int) int {
	value := defaultValue
	if fileValue > 0 {
		value = fileValue
	}
	if envValue := strings.TrimSpace(env(envKey)); envValue != "" {
		if parsed, err := strconv.Atoi(envValue); err == nil && parsed > 0 {
			value = parsed
		}
	}
	return value
}

func mergeDuration(defaultValue time.Duration, env func(string) string, envKey string, fileValue time.Duration) time.Duration {
	value := defaultValue
	if fileValue > 0 {
		value = fileValue
	}
	if envValue := strings.TrimSpace(env(envKey)); envValue != "" {
		if parsed, err := time.ParseDuration(envValue); err == nil && parsed > 0 {
			value = parsed
		}
	}
	return value
}

func mergeFloat(defaultValue float64, env func(string) string, envKey string, fileValue float64) float64 {
	value := defaultValue
	if fileValue > 0 {
		value = fileValue
	}
	if envValue := strings.TrimSpace(env(envKey)); envValue != "" {
		if parsed, err := strconv.ParseFloat(envValue, 64); err == nil && parsed > 0 {
			value = parsed
		}
	}
	return value
}

func mergeBool(defaultValue bool, env func(string) string, envKey string, fileValue *bool) (bool, error) {
	value := defaultValue
	if fileValue != nil {
		value = *fileValue
	}
	envValue := strings.TrimSpace(env(envKey))
	if envValue == "" {
		return value, nil
	}
	parsed, err := strconv.ParseBool(envValue)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", envKey, err)
	}
	return parsed, nil
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

func isReleaseMode(env func(string) string) bool {
	ginMode := strings.ToLower(strings.TrimSpace(env("GIN_MODE")))
	appEnvironment := strings.ToLower(strings.TrimSpace(env("HEIMDALLR_ENV")))
	return ginMode == "release" || appEnvironment == "production"
}
