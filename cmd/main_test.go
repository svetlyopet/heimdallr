package main

import (
	"io"
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/config"
	"github.com/svetlyopet/heimdallr/internal/http/server"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

func TestNewInjectorResolvesServer(t *testing.T) {
	cfg := config.DefaultTestConfig(io.Discard)
	cfg.Database.DatabaseURL = testutil.PostgresDatabaseURL(t)

	injector := newInjector(cfg)
	t.Cleanup(func() {
		_ = injector.Shutdown()
	})

	srv, err := do.Invoke[*server.Server](injector)
	require.NoError(t, err)
	require.NotNil(t, srv)
	require.NotNil(t, srv.HTTPHandler())
}
