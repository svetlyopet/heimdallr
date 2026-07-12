package token

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"github.com/svetlyopet/heimdallr/internal/token/api"
)

type stubTokenService struct {
	listResponse []api.Token
	listError    error
	createResp   api.TokenCreateResponse
	createError  error
	deleteError  error
}

func (s stubTokenService) List(_ context.Context) ([]api.Token, error) {
	if s.listError != nil {
		return nil, s.listError
	}

	return s.listResponse, nil
}

func (s stubTokenService) Create(_ context.Context, _ api.TokenCreateRequest, _ *uuid.UUID) (api.TokenCreateResponse, error) {
	if s.createError != nil {
		return api.TokenCreateResponse{}, s.createError
	}

	return s.createResp, nil
}

func (s stubTokenService) CreateSession(_ context.Context, _ string, _ []string, _ uuid.UUID) (api.TokenCreateResponse, error) {
	return api.TokenCreateResponse{}, nil
}

func (s stubTokenService) Delete(_ context.Context, _ string) error {
	return s.deleteError
}

func (s stubTokenService) Authenticate(_ context.Context, _ string) (authapi.AuthUser, error) {
	return authapi.AuthUser{Roles: []authapi.AuthRole{authapi.Admin}}, nil
}

func (s stubTokenService) RevokeSessionTokens(context.Context, string) error { return nil }
func (s stubTokenService) RevokeAllUserTokens(context.Context, string) error { return nil }

func newTokenRouter(t *testing.T, svc Service) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h, err := NewHandler(svc)
	require.NoError(t, err)

	r := gin.New()
	apiGroup := r.Group("/api")
	apiGroup.Use(testutil.AuthenticatedAdminMiddleware())
	RegisterRoutes(apiGroup, h, rbac.NewAuthorizer())

	return r
}

func TestHandlerCreateReturnsBadRequestForInvalidScopes(t *testing.T) {
	r := newTokenRouter(t, stubTokenService{createError: ErrInvalidScopes})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/auth/tokens", api.TokenCreateRequest{
		Name:   "bad-token",
		Scopes: []api.TokenScope{"invalid"},
	}, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerCreateReturnsCreated(t *testing.T) {
	tokenID := uuid.New()
	r := newTokenRouter(t, stubTokenService{
		createResp: api.TokenCreateResponse{
			Id:     tokenID,
			Name:   "ci-token",
			Scopes: []api.TokenScope{api.ApplicationWrite},
			Token:  "secret-token",
		},
	})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/auth/tokens", api.TokenCreateRequest{
		Name:   "ci-token",
		Scopes: []api.TokenScope{api.ApplicationWrite},
	}, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "ci-token", data["name"])
}

func TestHandlerDeleteReturnsNotFound(t *testing.T) {
	r := newTokenRouter(t, stubTokenService{deleteError: ErrTokenNotFound})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodDelete, "/api/v1/auth/tokens/"+uuid.New().String(), nil, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerListReturnsInternalServerError(t *testing.T) {
	r := newTokenRouter(t, stubTokenService{listError: errors.New("boom")})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/auth/tokens", nil, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}
