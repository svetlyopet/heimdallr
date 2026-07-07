package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/samber/do/v2"
	"github.com/svetlyopet/heimdallr/internal/config"
	"github.com/svetlyopet/heimdallr/internal/http/server"
	"github.com/svetlyopet/heimdallr/internal/logger"
)

func newInjector(cfg config.AppConfig) do.Injector {
	return do.New(
		config.ValuePackage(&cfg),
		config.InfrastructurePackage,
		server.Package,
	)
}

func run(ctx context.Context, cfg config.AppConfig) error {
	injector := newInjector(cfg)
	defer func() {
		_ = injector.Shutdown()
	}()

	srv := do.MustInvoke[*server.Server](injector)
	appLogger := do.MustInvoke[*logger.Logger](injector)

	appLogger.Info(
		ctx,
		"starting server",
		slog.String("host", cfg.Server.Host),
		slog.String("addr", ":"+cfg.Server.Port),
		slog.String("log_format", string(cfg.Logger.Format)),
		slog.String("log_level", cfg.Logger.Level.String()),
	)

	return srv.Run()
}

func main() {
	cfg, err := config.LoadFromFlags(os.Args[1:], os.Getenv)
	if err != nil {
		_, _ = os.Stderr.WriteString("failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	ctx := context.Background()
	if err = run(ctx, cfg); err != nil {
		appLogger := logger.New(cfg.Logger)
		appLogger.ErrorWithStack(ctx, "server stopped with error", err)
		os.Exit(1)
	}
}
