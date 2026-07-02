package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/auth"
)

type authServiceStub struct{}

func (authServiceStub) Authenticate(_ context.Context, username string, password string) (auth.GetResponse, error) {
	if username != "admin" || password != "AdminPassword123!" {
		return auth.GetResponse{}, auth.ErrInvalidCredentials
	}

	return auth.GetResponse{Username: username, Roles: []string{auth.RoleAdmin}}, nil
}

func (authServiceStub) List(context.Context) ([]auth.GetResponse, error) {
	return nil, errors.New("not implemented")
}

func (authServiceStub) Create(context.Context, auth.CreateRequest) (auth.GetResponse, error) {
	return auth.GetResponse{}, errors.New("not implemented")
}

func (authServiceStub) Update(context.Context, string, auth.UpdateRequest) (auth.GetResponse, error) {
	return auth.GetResponse{}, errors.New("not implemented")
}

func (authServiceStub) Delete(context.Context, string) error {
	return errors.New("not implemented")
}

func (authServiceStub) EnsureRootUser(context.Context) (string, error) {
	return "", errors.New("not implemented")
}

func (authServiceStub) HasAnyRole(_ auth.GetResponse, _ ...string) bool {
	return false
}

func TestAuthenticationRejectsMissingHeaders(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Authentication(authServiceStub{}))
	r.GET("/secure", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticationRejectsInvalidCredentials(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Authentication(authServiceStub{}))
	r.GET("/secure", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set(authHeaderUsername, "ghost")
	req.Header.Set(authHeaderPassword, "wrong")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticationAcceptsValidCredentials(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Authentication(authServiceStub{}))
	r.GET("/secure", func(ctx *gin.Context) {
		user, exists := ctx.Get("auth.user")
		require.True(t, exists)

		authUser, ok := user.(auth.GetResponse)
		require.True(t, ok)
		require.Equal(t, "admin", authUser.Username)

		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set(authHeaderUsername, "admin")
	req.Header.Set(authHeaderPassword, "AdminPassword123!")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}
