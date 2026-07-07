package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/ditest"
)

func TestServerHTTPHandlerHealthEndpoint(t *testing.T) {
	injector := ditest.NewServerInjector(t)
	srv := ditest.MustInvokeServer(t, injector)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Host = "localhost"
	rec := httptest.NewRecorder()

	srv.HTTPHandler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"ok"`)
}
