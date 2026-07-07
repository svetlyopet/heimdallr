package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/logger"
)

func TestNewWritesJSONLogs(t *testing.T) {
	var output bytes.Buffer
	appLogger := logger.New(logger.Config{
		Format: logger.FormatJSON,
		Level:  slog.LevelInfo,
		Output: &output,
	})

	appLogger.Info(context.Background(), "hello", slog.String("component", "test"))

	require.Contains(t, output.String(), `"msg":"hello"`)
	require.Contains(t, output.String(), `"component":"test"`)
}

func TestNewWritesTextLogsAtConfiguredLevel(t *testing.T) {
	var output bytes.Buffer
	appLogger := logger.New(logger.Config{
		Format: logger.FormatText,
		Level:  slog.LevelWarn,
		Output: &output,
	})

	appLogger.Info(context.Background(), "hidden")
	appLogger.Warn(context.Background(), "visible")

	logOutput := output.String()
	require.NotContains(t, logOutput, "hidden")
	require.Contains(t, logOutput, "visible")
	require.True(t, strings.Contains(logOutput, "level=WARN") || strings.Contains(logOutput, "WARN"))
}
