package rbac_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

func TestAuthorizerHasScope(t *testing.T) {
	t.Parallel()

	authorizer := rbac.NewAuthorizer()

	reader := authapi.AuthUser{Roles: []authapi.AuthRole{authapi.Reader, authapi.AuthRole(rbac.ScopeRead)}}
	require.True(t, authorizer.HasScope(reader, rbac.ScopeRead))
	require.False(t, authorizer.HasScope(reader, rbac.ScopeApplicationWrite))

	appWriter := authapi.AuthUser{Roles: []authapi.AuthRole{authapi.Reader, authapi.AuthRole(rbac.ScopeApplicationWrite)}}
	require.True(t, authorizer.HasScope(appWriter, rbac.ScopeApplicationWrite))
	require.False(t, authorizer.HasScope(appWriter, rbac.ScopeAutomationWrite))

	admin := authapi.AuthUser{Roles: []authapi.AuthRole{authapi.Admin, authapi.Reader}}
	require.True(t, authorizer.HasScope(admin, rbac.ScopeAutomationWrite))
}

func TestAuthorizerHasAnyRoleEmptyDenies(t *testing.T) {
	t.Parallel()

	authorizer := rbac.NewAuthorizer()
	user := authapi.AuthUser{Roles: []authapi.AuthRole{authapi.Admin}}

	require.False(t, authorizer.HasAnyRole(user))
}

func TestRequireRoleMiddleware(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	authorizer := rbac.NewAuthorizer()
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set("auth.user", authapi.AuthUser{Roles: []authapi.AuthRole{authapi.Reader}})
		ctx.Next()
	})
	router.GET("/secure", rbac.RequireRole(authorizer, rbac.RoleAdmin), func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestStrictScopeMiddleware(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	authorizer := rbac.NewAuthorizer()
	policies := map[string]string{"CreateApplication": rbac.ScopeApplicationWrite}
	handler := func(ctx *gin.Context, request any) (any, error) {
		return "ok", nil
	}
	wrapped := rbac.StrictScopeMiddleware(authorizer, policies)(handler, "CreateApplication")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("auth.user", authapi.AuthUser{Roles: []authapi.AuthRole{authapi.Reader, authapi.AuthRole(rbac.ScopeRead)}})

	_, err := wrapped(ctx, nil)
	require.Error(t, err)

	var httpErr *rbac.HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, 403, httpErr.Status)
}

func TestScopesToRoles(t *testing.T) {
	t.Parallel()

	require.Equal(t, []string{rbac.RoleAdmin, rbac.RoleReader}, rbac.ScopesToRoles([]string{rbac.ScopeAdmin}))
	require.Equal(t, []string{rbac.RoleReader, rbac.ScopeRead}, rbac.ScopesToRoles([]string{rbac.ScopeRead}))
}
