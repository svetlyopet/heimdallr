package config_test

import (
	"testing"

	"log/slog"

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
	require.Equal(t, constants.AppDefaultName+".db", cfg.Database.DatabasePath)
	require.Equal(t, logger.FormatText, cfg.Logger.Format)
	require.Equal(t, slog.LevelInfo, cfg.Logger.Level)
	require.Empty(t, cfg.Auth.BootstrapRootPassword)
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
		[]string{"-log-format", "json", "-log-level", "debug", "-server-name", "testhost", "-server-port", "9090", "-database-path", "custom.db"},
		env,
	)
	require.NoError(t, err)

	require.Equal(t, "testhost", cfg.Server.Host)
	require.Equal(t, "9090", cfg.Server.Port)
	require.Equal(t, "postgres://example", cfg.Database.DatabaseURL)
	require.Equal(t, "custom.db", cfg.Database.DatabasePath)
	require.Equal(t, logger.FormatJSON, cfg.Logger.Format)
	require.Equal(t, slog.LevelDebug, cfg.Logger.Level)
	require.Equal(t, "BootstrapPassword12!", cfg.Auth.BootstrapRootPassword)
}
