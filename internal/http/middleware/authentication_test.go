package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/token"
	"github.com/svetlyopet/heimdallr/internal/token/api"
)

type tokenServiceStub struct {
	token string
	err   error
}

func (s tokenServiceStub) List(context.Context) ([]api.Token, error) {
	return nil, token.ErrInvalidToken
}

func (s tokenServiceStub) Create(context.Context, api.TokenCreateRequest, *uuid.UUID) (api.TokenCreateResponse, error) {
	return api.TokenCreateResponse{}, token.ErrInvalidToken
}

func (s tokenServiceStub) CreateSession(context.Context, string, []string, uuid.UUID) (api.TokenCreateResponse, error) {
	return api.TokenCreateResponse{}, token.ErrInvalidToken
}

func (s tokenServiceStub) Delete(context.Context, string) error {
	return token.ErrInvalidToken
}

func (s tokenServiceStub) RevokeSessionTokens(context.Context, string) error { return nil }
func (s tokenServiceStub) RevokeAllUserTokens(context.Context, string) error   { return nil }

func (s tokenServiceStub) Authenticate(_ context.Context, plainToken string) (authapi.AuthUser, error) {
	if s.err != nil {
		return authapi.AuthUser{}, s.err
	}

	if plainToken != s.token {
		return authapi.AuthUser{}, token.ErrInvalidToken
	}

	return authapi.AuthUser{Username: "token-user", Roles: []authapi.AuthRole{authapi.Admin}}, nil
}

func TestAuthenticationRejectsMissingBearerToken(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Authentication(tokenServiceStub{token: "valid-token"}))
	r.GET("/secure", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticationRejectsInvalidBearerToken(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Authentication(tokenServiceStub{token: "valid-token"}))
	r.GET("/secure", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set(authHeaderBearer, "Bearer wrong-token")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticationAcceptsValidBearerToken(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Authentication(tokenServiceStub{token: "valid-token"}))
	r.GET("/secure", func(ctx *gin.Context) {
		user, exists := ctx.Get("auth.user")
		require.True(t, exists)

		authUser, ok := user.(authapi.AuthUser)
		require.True(t, ok)
		require.Equal(t, "token-user", authUser.Username)

		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set(authHeaderBearer, "Bearer valid-token")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthenticationRejectsMalformedAuthorizationHeader(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Authentication(tokenServiceStub{token: "valid-token"}))
	r.GET("/secure", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set(authHeaderBearer, "Token valid-token")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
