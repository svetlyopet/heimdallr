package rbac_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

func TestStrictHandlerErrorFuncReturnsGenericInternalServerError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/automations", nil)
	ctx.Set(rbac.OperationIDKey, "ListAutomations")

	dbErr := errors.New(`pq: relation "automations" does not exist at /var/lib/postgres/data`)

	errorFunc := rbac.NewStrictHandlerErrorFunc(logger.NewText())
	errorFunc(ctx, dbErr)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"Internal Server Error"`)
	require.NotContains(t, recorder.Body.String(), "automations")
	require.NotContains(t, recorder.Body.String(), "/var/lib/postgres")
}

func TestStrictHandlerErrorFuncPreservesHTTPError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/automations", nil)

	errorFunc := rbac.NewStrictHandlerErrorFunc(logger.NewText())
	errorFunc(ctx, &rbac.HTTPError{
		Status:  http.StatusForbidden,
		Message: "forbidden",
	})

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"forbidden"`)
}

func TestStrictHandlerErrorFuncDoesNotExposeWrappedCause(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/jobs", nil)

	wrapped := errors.Join(errors.New("sql: connection refused"), errors.New("stack frame /internal/job/repository.go"))

	errorFunc := rbac.NewStrictHandlerErrorFunc(logger.NewText())
	errorFunc(ctx, wrapped)

	body := recorder.Body.String()
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.True(t, strings.Contains(body, `"Internal Server Error"`))
	require.NotContains(t, body, "connection refused")
	require.NotContains(t, body, "repository.go")
}
