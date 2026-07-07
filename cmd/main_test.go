package main

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/config"
	"github.com/svetlyopet/heimdallr/internal/http/server"
)

func TestNewInjectorResolvesServer(t *testing.T) {
	cfg := config.DefaultTestConfig(io.Discard)
	cfg.Database.DatabasePath = filepath.Join(t.TempDir(), "heimdallr.db")

	injector := newInjector(cfg)
	t.Cleanup(func() {
		_ = injector.Shutdown()
	})

	srv, err := do.Invoke[*server.Server](injector)
	require.NoError(t, err)
	require.NotNil(t, srv)
	require.NotNil(t, srv.HTTPHandler())
}
