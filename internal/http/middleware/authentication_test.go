package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/auth"
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
func (s tokenServiceStub) RevokeAllUserTokens(context.Context, string) error { return nil }
func (s tokenServiceStub) RevokeSessionToken(context.Context, string) error  { return nil }

func (s tokenServiceStub) Authenticate(_ context.Context, plainToken string) (authapi.AuthUser, error) {
	if s.err != nil {
		return authapi.AuthUser{}, s.err
	}

	if plainToken != s.token {
		return authapi.AuthUser{}, token.ErrInvalidToken
	}

	return authapi.AuthUser{Username: "token-user", Roles: []authapi.AuthRole{authapi.Admin}}, nil
}

func (s tokenServiceStub) AuthenticateSession(ctx context.Context, plainToken string) (authapi.AuthUser, error) {
	return s.Authenticate(ctx, plainToken)
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
		requestUser, err := auth.UserFromContext(ctx.Request.Context())
		require.NoError(t, err)
		require.Equal(t, authUser, requestUser)
		mechanism, credential, err := auth.AuthenticationFromContext(ctx.Request.Context())
		require.NoError(t, err)
		require.Equal(t, auth.AuthenticationBearer, mechanism)
		require.Equal(t, "valid-token", credential)

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

func TestAuthenticationAcceptsCookieAndRequiresCSRFForUnsafeMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Authentication(tokenServiceStub{token: "session-token"}))
	r.GET("/secure", func(ctx *gin.Context) {
		mechanism, _, err := auth.AuthenticationFromContext(ctx.Request.Context())
		require.NoError(t, err)
		require.Equal(t, auth.AuthenticationCookie, mechanism)
		ctx.Status(http.StatusOK)
	})
	r.POST("/secure", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	getRequest := httptest.NewRequest(http.MethodGet, "/secure", nil)
	getRequest.AddCookie(&http.Cookie{Name: "heimdallr_session", Value: "session-token"})
	getResponse := httptest.NewRecorder()
	r.ServeHTTP(getResponse, getRequest)
	require.Equal(t, http.StatusOK, getResponse.Code)

	missingCSRFRequest := httptest.NewRequest(http.MethodPost, "/secure", nil)
	missingCSRFRequest.AddCookie(&http.Cookie{Name: "heimdallr_session", Value: "session-token"})
	missingCSRFResponse := httptest.NewRecorder()
	r.ServeHTTP(missingCSRFResponse, missingCSRFRequest)
	require.Equal(t, http.StatusForbidden, missingCSRFResponse.Code)

	validCSRFRequest := httptest.NewRequest(http.MethodPost, "/secure", nil)
	validCSRFRequest.AddCookie(&http.Cookie{Name: "heimdallr_session", Value: "session-token"})
	validCSRFRequest.AddCookie(&http.Cookie{Name: "heimdallr_csrf", Value: "csrf-token"})
	validCSRFRequest.Header.Set("X-CSRF-Token", "csrf-token")
	validCSRFResponse := httptest.NewRecorder()
	r.ServeHTTP(validCSRFResponse, validCSRFRequest)
	require.Equal(t, http.StatusOK, validCSRFResponse.Code)
}

func TestAuthenticationBearerUnsafeRequestDoesNotRequireCSRF(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Authentication(tokenServiceStub{token: "valid-token"}))
	r.POST("/secure", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/secure", nil)
	req.Header.Set(authHeaderBearer, "Bearer valid-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthenticationAllowsIdempotentOptionalLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Authentication(tokenServiceStub{token: "valid-token"}, AuthenticationConfig{
		OptionalPaths: []string{"/api/v1/auth/logout"},
	}))
	r.POST("/api/v1/auth/logout", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil),
		func() *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
			req.AddCookie(&http.Cookie{Name: "heimdallr_session", Value: "expired-session"})
			return req
		}(),
	} {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		require.Equal(t, http.StatusNoContent, rr.Code)
	}
}
