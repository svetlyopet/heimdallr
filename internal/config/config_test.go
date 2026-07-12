package config_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/config"
	"github.com/svetlyopet/heimdallr/internal/constants"
	"github.com/svetlyopet/heimdallr/internal/logger"
)

func TestLoadFromFlagsDefaults(t *testing.T) {
	cfg, err := config.LoadFromFlags(nil, func(string) string { return "" })
	require.NoError(t, err)

	require.Equal(t, constants.ApiDefaultHost, cfg.Server.Host)
	require.Equal(t, constants.ApiDefaultPort, cfg.Server.Port)
	require.Empty(t, cfg.Database.DatabaseURL)
	require.Equal(t, logger.FormatText, cfg.Logger.Format)
	require.Equal(t, slog.LevelInfo, cfg.Logger.Level)
	require.Empty(t, cfg.Auth.BootstrapRootPassword)
	require.Equal(t, 5*time.Second, cfg.Server.ReadHeaderTimeout)
	require.Equal(t, 5<<20, int(cfg.Server.MaxRequestBodyBytes))
	require.Equal(t, 4<<20, int(cfg.Server.MaxDecodedOutputBytes))
	require.Equal(t, 100, cfg.Server.MaxPaginationLimit)
	require.Equal(t, 90*24*time.Hour, cfg.Auth.DefaultAPITokenTTL)
	require.Equal(t, 365*24*time.Hour, cfg.Auth.MaxAPITokenTTL)
	require.False(t, cfg.Auth.CookieSecure)
}

func TestLoadFromFlagsRequiresSecureCookiesInReleaseMode(t *testing.T) {
	env := func(key string) string {
		switch key {
		case "GIN_MODE":
			return "release"
		case "HEIMDALLR_COOKIE_SECURE":
			return "false"
		default:
			return ""
		}
	}

	_, err := config.LoadFromFlags(nil, env)
	require.Error(t, err)
}

func TestLoadFromFlagsReadsEnvironment(t *testing.T) {
	env := func(key string) string {
		switch key {
		case "DATABASE_URL":
			return "postgres://example"
		case "HEIMDALLR_BOOTSTRAP_ROOT_PASSWORD":
			return "BootstrapPassword12!"
		default:
			return ""
		}
	}

	cfg, err := config.LoadFromFlags(
		[]string{"-log-format", "json", "-log-level", "debug", "-server-name", "testhost", "-server-port", "9090"},
		env,
	)
	require.NoError(t, err)

	require.Equal(t, "testhost", cfg.Server.Host)
	require.Equal(t, "9090", cfg.Server.Port)
	require.Equal(t, "postgres://example", cfg.Database.DatabaseURL)
	require.Equal(t, logger.FormatJSON, cfg.Logger.Format)
	require.Equal(t, slog.LevelDebug, cfg.Logger.Level)
	require.Equal(t, "BootstrapPassword12!", cfg.Auth.BootstrapRootPassword)
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoadFromFlagsReadsConfigFile(t *testing.T) {
	path := writeConfigFile(t, `
database:
  url: postgres://file
server:
  host: filehost
  port: "9090"
logger:
  format: json
  level: warn
auth:
  bootstrap_root_password: FilePassword12!
  login_rate_limit_max: 25
`)

	cfg, err := config.LoadFromFlags([]string{"-config", path}, func(string) string { return "" })
	require.NoError(t, err)

	require.Equal(t, "postgres://file", cfg.Database.DatabaseURL)
	require.Equal(t, "filehost", cfg.Server.Host)
	require.Equal(t, "9090", cfg.Server.Port)
	require.Equal(t, logger.FormatJSON, cfg.Logger.Format)
	require.Equal(t, slog.LevelWarn, cfg.Logger.Level)
	require.Equal(t, "FilePassword12!", cfg.Auth.BootstrapRootPassword)
	require.Equal(t, 25, cfg.Auth.LoginRateLimitMax)
}

func TestLoadFromFlagsEnvironmentOverridesConfigFile(t *testing.T) {
	path := writeConfigFile(t, `
database:
  url: postgres://file
server:
  port: "9090"
auth:
  bootstrap_root_password: FilePassword12!
`)

	env := func(key string) string {
		switch key {
		case "DATABASE_URL":
			return "postgres://env"
		case "HEIMDALLR_BOOTSTRAP_ROOT_PASSWORD":
			return "EnvPassword12!"
		default:
			return ""
		}
	}

	cfg, err := config.LoadFromFlags([]string{"-config", path}, env)
	require.NoError(t, err)

	require.Equal(t, "postgres://env", cfg.Database.DatabaseURL)
	require.Equal(t, "9090", cfg.Server.Port)
	require.Equal(t, "EnvPassword12!", cfg.Auth.BootstrapRootPassword)
}

func TestLoadFromFlagsExplicitFlagOverridesConfigFileAndEnvironment(t *testing.T) {
	path := writeConfigFile(t, `
server:
  host: filehost
  port: "9090"
logger:
  format: json
  level: debug
`)

	env := func(key string) string {
		return ""
	}

	cfg, err := config.LoadFromFlags(
		[]string{
			"-config", path,
			"-server-name", "flaghost",
			"-server-port", "7070",
			"-log-format", "text",
			"-log-level", "error",
		},
		env,
	)
	require.NoError(t, err)

	require.Equal(t, "flaghost", cfg.Server.Host)
	require.Equal(t, "7070", cfg.Server.Port)
	require.Equal(t, logger.FormatText, cfg.Logger.Format)
	require.Equal(t, slog.LevelError, cfg.Logger.Level)
}

func TestLoadFromFlagsConfigFilePortUsedWhenFlagNotExplicit(t *testing.T) {
	path := writeConfigFile(t, `
server:
  port: "9090"
`)

	cfg, err := config.LoadFromFlags([]string{"-config", path}, func(string) string { return "" })
	require.NoError(t, err)
	require.Equal(t, "9090", cfg.Server.Port)
}

func TestLoadFromFlagsMissingConfigFileReturnsError(t *testing.T) {
	_, err := config.LoadFromFlags(
		[]string{"-config", filepath.Join(t.TempDir(), "missing.yaml")},
		func(string) string { return "" },
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "read config file")
}

func TestLoadFromFlagsInvalidConfigFileReturnsError(t *testing.T) {
	path := writeConfigFile(t, "database:\n  url: [")
	_, err := config.LoadFromFlags([]string{"-config", path}, func(string) string { return "" })
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse config file")
}

func TestLoadFromFlagsRequiresSecureCookiesInReleaseModeWithConfigFile(t *testing.T) {
	path := writeConfigFile(t, `
auth:
  cookie_secure: false
`)

	env := func(key string) string {
		if key == "GIN_MODE" {
			return "release"
		}
		return ""
	}

	_, err := config.LoadFromFlags([]string{"-config", path}, env)
	require.Error(t, err)
}
