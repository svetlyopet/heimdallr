package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type tokenServiceStub struct{}

func (tokenServiceStub) Authenticate(context.Context, string) (api.AuthUser, error) {
	return api.AuthUser{}, nil
}

func (tokenServiceStub) CreateSession(context.Context, SessionTokenCreateRequest, uuid.UUID) (SessionTokenCreateResponse, error) {
	return SessionTokenCreateResponse{}, nil
}

func (tokenServiceStub) RevokeSessionTokens(context.Context, string) error { return nil }
func (tokenServiceStub) RevokeAllUserTokens(context.Context, string) error { return nil }
func (tokenServiceStub) RevokeSessionToken(context.Context, string) error  { return nil }

func TestProtectedUserRoutesRequireAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc, _, _ := newTestService(t, ServiceConfig{})
	h, err := NewHandler(svc, tokenServiceStub{})
	require.NoError(t, err)

	router := gin.New()
	apiGroup := router.Group("/api")
	apiGroup.Use(testutil.AuthenticatedReaderMiddleware())
	RegisterProtectedRoutes(apiGroup, h, rbac.NewAuthorizer(), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestHasAnyRoleEmptyRequiredRoles(t *testing.T) {
	svc, _, _ := newTestService(t, ServiceConfig{})
	user := api.AuthUser{Roles: []api.AuthRole{api.Admin}}
	require.False(t, svc.HasAnyRole(user))
}

func TestLegacyPasswordRejectedWhenResetRequired(t *testing.T) {
	svc, _, db := newTestService(t, ServiceConfig{})

	legacyHash := legacyHashForTest("LegacyPassword123!")
	require.NoError(t, db.Create(&User{
		Username:              "legacy-user",
		Email:                 "legacy@example.com",
		PasswordHash:          legacyHash,
		PasswordResetRequired: true,
		Roles:                 []string{RoleReader},
	}).Error)

	_, err := svc.Authenticate(t.Context(), "legacy-user", "LegacyPassword123!")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestCreateUserHashesWithBcrypt(t *testing.T) {
	svc, repo, _ := newTestService(t, ServiceConfig{})

	created, err := svc.Create(t.Context(), api.AuthCreateUserRequest{
		Username: "bcrypt-user",
		Email:    openapi_types.Email("bcrypt@example.com"),
		Password: "StrongPassword123!",
	})
	require.NoError(t, err)

	stored, err := repo.FindByID(t.Context(), created.Id)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(stored.PasswordHash, "$2"))
	valid, needsRehash := verifyPassword("StrongPassword123!", stored.PasswordHash)
	require.True(t, valid)
	require.False(t, needsRehash)
}
