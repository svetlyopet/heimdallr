package ditest

import (
	"bytes"
	"testing"

	"github.com/samber/do/v2"
	"github.com/svetlyopet/heimdallr/internal/app"
	"github.com/svetlyopet/heimdallr/internal/config"
	"github.com/svetlyopet/heimdallr/internal/http/server"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type Option func(do.Injector)

func WithConfig(cfg config.AppConfig) Option {
	return func(i do.Injector) {
		do.OverrideValue(i, &cfg)
	}
}

func defaultTestConfig(t *testing.T) config.AppConfig {
	t.Helper()

	cfg := config.DefaultTestConfig(bytes.NewBuffer(nil))
	cfg.Database.DatabaseURL = testutil.PostgresDatabaseURL(t)

	return cfg
}

func NewInjector(t *testing.T, opts ...Option) do.Injector {
	t.Helper()

	cfg := defaultTestConfig(t)
	injector := do.New(
		config.ValuePackage(&cfg),
		config.InfrastructurePackage,
		app.Package,
	)

	t.Cleanup(func() {
		_ = injector.Shutdown()
	})

	for _, opt := range opts {
		opt(injector)
	}

	return injector
}

func NewServerInjector(t *testing.T, opts ...Option) do.Injector {
	t.Helper()

	cfg := defaultTestConfig(t)
	injector := do.New(
		config.ValuePackage(&cfg),
		config.InfrastructurePackage,
		server.Package,
	)

	t.Cleanup(func() {
		_ = injector.Shutdown()
	})

	for _, opt := range opts {
		opt(injector)
	}

	return injector
}

func MustInvokeApp(t *testing.T, injector do.Injector) *app.App {
	t.Helper()

	application, err := do.Invoke[*app.App](injector)
	if err != nil {
		t.Fatalf("invoke app: %v", err)
	}

	return application
}

func MustInvokeServer(t *testing.T, injector do.Injector) *server.Server {
	t.Helper()

	srv, err := do.Invoke[*server.Server](injector)
	if err != nil {
		t.Fatalf("invoke server: %v", err)
	}

	return srv
}
