package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/svetlyopet/heimdallr/internal/constants"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/http/server"
	"github.com/svetlyopet/heimdallr/internal/logger"
)

func main() {
	logFormat := flag.String("log-format", "text", "log format: text or json")
	logLevel := flag.String("log-level", "info", "log level: debug, info, warn, or error")
	serverName := flag.String("server-name", constants.ApiDefaultHost, "server name")
	serverPort := flag.String("server-port", constants.ApiDefaultPort, "server port")
	databasePath := flag.String("database-path", constants.AppDefaultName+".db", "sqlite database path when DATABASE_URL is unset")
	flag.Parse()

	appLogger := logger.New(logger.Config{
		Format: logger.Format(*logFormat),
		Level:  parseLogLevel(*logLevel),
		Output: os.Stdout,
	})

	ctx := context.Background()

	db, err := database.Open(database.Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		DatabasePath: *databasePath,
	})
	if err != nil {
		appLogger.ErrorWithStack(ctx, "failed to initialize database", err)
		os.Exit(1)
	}

	srv, err := server.NewServer(*serverName, *serverPort, db, appLogger)
	if err != nil {
		appLogger.ErrorWithStack(ctx, "failed to initialize server", err)
		os.Exit(1)
	}

	appLogger.Info(
		ctx,
		"starting server",
		slog.String("host", "localhost"),
		slog.String("addr", ":"+*serverPort),
		slog.String("log_format", *logFormat),
		slog.String("log_level", *logLevel),
	)

	if err = srv.Run(); err != nil {
		appLogger.ErrorWithStack(ctx, "server stopped with error", err)
		os.Exit(1)
	}
}

func parseLogLevel(level string) slog.Level {
	switch level {
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
