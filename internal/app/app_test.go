package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/ditest"
)

func TestAppBootstrapCreatesRootUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	injector := ditest.NewInjector(t)
	application := ditest.MustInvokeApp(t, injector)

	require.NoError(t, application.Bootstrap(t.Context()))

	_, err := application.AuthService().Authenticate(t.Context(), "root", "IntegrationTestPassword12!")
	require.NoError(t, err)
}

func TestAppRegisterRoutesMountsHealthThroughServer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	injector := ditest.NewServerInjector(t)
	srv := ditest.MustInvokeServer(t, injector)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Host = "localhost"
	rec := httptest.NewRecorder()

	srv.HTTPHandler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"ok"`)
}
