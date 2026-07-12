package app_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedStrictHandlersUseHTTPRequestContext(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "*", "api", "api.gen.go"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		generated, readErr := os.ReadFile(file)
		require.NoError(t, readErr)

		strictCalls := bytes.Count(generated, []byte("return sh.ssi."))
		requestContextCalls := bytes.Count(generated, []byte("(ctx.Request.Context(), request.("))
		require.Positive(t, strictCalls, file)
		require.Equal(t, strictCalls, requestContextCalls, file)
	}
}
