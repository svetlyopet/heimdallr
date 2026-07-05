package token

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubTokenService struct {
	listResponse []GetResponse
	listError    error
	createResp   CreateResponse
	createError  error
	deleteError  error
}

func (s stubTokenService) List(_ context.Context) ([]GetResponse, error) {
	if s.listError != nil {
		return nil, s.listError
	}

	return s.listResponse, nil
}

func (s stubTokenService) Create(_ context.Context, _ CreateRequest, _ *uuid.UUID) (CreateResponse, error) {
	if s.createError != nil {
		return CreateResponse{}, s.createError
	}

	return s.createResp, nil
}

func (s stubTokenService) Delete(_ context.Context, _ string) error {
	return s.deleteError
}

func (s stubTokenService) Authenticate(_ context.Context, _ string) (auth.GetResponse, error) {
	return auth.GetResponse{}, nil
}

func (s stubTokenService) HasScope(_ auth.GetResponse, _ string) bool {
	return true
}

func newTokenRouter(t *testing.T, svc Service) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h, err := NewHandler(svc)
	require.NoError(t, err)

	r := gin.New()
	api := r.Group("/api")
	RegisterRoutes(api, h, authServiceStub{})

	return r
}

type authServiceStub struct{}

func (authServiceStub) Authenticate(context.Context, string, string) (auth.GetResponse, error) {
	return auth.GetResponse{Roles: []string{auth.RoleAdmin}}, nil
}

func (authServiceStub) List(context.Context) ([]auth.GetResponse, error) { return nil, nil }
func (authServiceStub) Create(context.Context, auth.CreateRequest) (auth.GetResponse, error) {
	return auth.GetResponse{}, nil
}
func (authServiceStub) Update(context.Context, string, auth.UpdateRequest) (auth.GetResponse, error) {
	return auth.GetResponse{}, nil
}
func (authServiceStub) Delete(context.Context, string) error { return nil }
func (authServiceStub) EnsureRootUser(context.Context) (string, error) { return "", nil }
func (authServiceStub) HasAnyRole(auth.GetResponse, ...string) bool { return true }

func TestHandlerCreateReturnsBadRequestForInvalidScopes(t *testing.T) {
	r := newTokenRouter(t, stubTokenService{createError: ErrInvalidScopes})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/auth/tokens", CreateRequest{
		Name:   "bad-token",
		Scopes: []string{"invalid"},
	}, testutil.AuthHeaders("admin", "password"))
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerCreateReturnsCreated(t *testing.T) {
	tokenID := uuid.New()
	r := newTokenRouter(t, stubTokenService{
		createResp: CreateResponse{
			GetResponse: GetResponse{ID: tokenID, Name: "ci-token", Scopes: []string{ScopeApplicationWrite}},
			Token:       "secret-token",
		},
	})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/auth/tokens", CreateRequest{
		Name:   "ci-token",
		Scopes: []string{ScopeApplicationWrite},
	}, testutil.AuthHeaders("admin", "password"))
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "ci-token", data["name"])
}

func TestHandlerDeleteReturnsNotFound(t *testing.T) {
	r := newTokenRouter(t, stubTokenService{deleteError: ErrTokenNotFound})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodDelete, "/api/v1/auth/tokens/"+uuid.New().String(), nil, testutil.AuthHeaders("admin", "password"))
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerListReturnsInternalServerError(t *testing.T) {
	r := newTokenRouter(t, stubTokenService{listError: errors.New("boom")})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/auth/tokens", nil, testutil.AuthHeaders("admin", "password"))
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}
