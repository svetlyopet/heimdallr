package token

import (
	"context"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/auth"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

func newAuthServiceWithTokenRepo(t *testing.T) (auth.Service, auth.Repository, Service) {
	t.Helper()

	db := testutil.NewPostgresDB(t)
	authRepo := auth.NewRepository(db)
	tokenRepo := NewRepository(db)
	tokenSvc := NewService(tokenRepo, authRepo, nil, DefaultServiceConfig())
	authSvc := auth.NewService(authRepo, NewAuthTokenRepository(tokenRepo), db, nil, auth.ServiceConfig{})

	return authSvc, authRepo, tokenSvc
}

func authenticatedContext(t *testing.T, user authapi.AuthUser, credential string) context.Context {
	t.Helper()

	return auth.ContextWithAuthentication(t.Context(), user, auth.AuthenticationBearer, credential)
}

func parseUserID(t *testing.T, userID string) uuid.UUID {
	t.Helper()

	id, err := uuid.Parse(userID)
	require.NoError(t, err)
	return id
}

func TestAuthUpdateRootPasswordChangePreservesCurrentSession(t *testing.T) {
	authSvc, authRepo, tokenSvc := newAuthServiceWithTokenRepo(t)

	_, err := authSvc.EnsureRootUser(t.Context())
	require.NoError(t, err)

	root, err := authRepo.FindByUsername(t.Context(), "root")
	require.NoError(t, err)

	currentSession, err := tokenSvc.CreateSession(t.Context(), "session-root-current", []string{rbac.ScopeAdmin}, root.ID)
	require.NoError(t, err)

	otherSession, err := tokenSvc.CreateSession(t.Context(), "session-root-other", []string{rbac.ScopeAdmin}, root.ID)
	require.NoError(t, err)

	newPassword := "NewRootPassword123!"
	rootUser := authapi.AuthUser{
		Id:       root.ID.String(),
		Username: root.Username,
		Email:    openapi_types.Email(root.Email),
		Roles:    []authapi.AuthRole{authapi.Admin},
	}
	rootCtx := authenticatedContext(t, rootUser, currentSession.Token)
	_, err = authSvc.Update(rootCtx, root.ID.String(), authapi.AuthUpdateUserRequest{Password: &newPassword})
	require.NoError(t, err)

	_, err = tokenSvc.AuthenticateSession(t.Context(), currentSession.Token)
	require.NoError(t, err)

	_, err = tokenSvc.AuthenticateSession(t.Context(), otherSession.Token)
	require.Error(t, err)
}

func TestAuthUpdateAdminPasswordChangePreservesCurrentSession(t *testing.T) {
	authSvc, _, tokenSvc := newAuthServiceWithTokenRepo(t)

	admin, err := authSvc.Create(t.Context(), authapi.AuthCreateUserRequest{
		Username: "self-admin",
		Email:    openapi_types.Email("self-admin@example.com"),
		Password: "StrongPassword123!",
		Roles:    &[]authapi.AuthRole{authapi.Admin},
	})
	require.NoError(t, err)

	adminID := parseUserID(t, admin.Id)

	currentSession, err := tokenSvc.CreateSession(t.Context(), "session-admin-current", []string{rbac.ScopeAdmin}, adminID)
	require.NoError(t, err)

	otherSession, err := tokenSvc.CreateSession(t.Context(), "session-admin-other", []string{rbac.ScopeAdmin}, adminID)
	require.NoError(t, err)

	newPassword := "NewAdminPassword123!"
	adminCtx := authenticatedContext(t, admin, currentSession.Token)
	_, err = authSvc.Update(adminCtx, admin.Id, authapi.AuthUpdateUserRequest{Password: &newPassword})
	require.NoError(t, err)

	_, err = tokenSvc.AuthenticateSession(t.Context(), currentSession.Token)
	require.NoError(t, err)

	_, err = tokenSvc.AuthenticateSession(t.Context(), otherSession.Token)
	require.Error(t, err)
}

func TestAuthUpdateAdminPasswordChangeRevokesAllForOtherUser(t *testing.T) {
	authSvc, _, tokenSvc := newAuthServiceWithTokenRepo(t)

	admin, err := authSvc.Create(t.Context(), authapi.AuthCreateUserRequest{
		Username: "reset-admin",
		Email:    openapi_types.Email("reset-admin@example.com"),
		Password: "StrongPassword123!",
		Roles:    &[]authapi.AuthRole{authapi.Admin},
	})
	require.NoError(t, err)

	target, err := authSvc.Create(t.Context(), authapi.AuthCreateUserRequest{
		Username: "target-user",
		Email:    openapi_types.Email("target-user@example.com"),
		Password: "StrongPassword123!",
		Roles:    &[]authapi.AuthRole{authapi.Reader},
	})
	require.NoError(t, err)

	targetID := parseUserID(t, target.Id)

	firstSession, err := tokenSvc.CreateSession(t.Context(), "session-target-one", []string{rbac.ScopeRead}, targetID)
	require.NoError(t, err)

	secondSession, err := tokenSvc.CreateSession(t.Context(), "session-target-two", []string{rbac.ScopeRead}, targetID)
	require.NoError(t, err)

	newPassword := "ResetTargetPassword123!"
	adminCtx := authenticatedContext(t, admin, "admin-reset-token")
	_, err = authSvc.Update(adminCtx, target.Id, authapi.AuthUpdateUserRequest{Password: &newPassword})
	require.NoError(t, err)

	_, err = tokenSvc.AuthenticateSession(t.Context(), firstSession.Token)
	require.Error(t, err)

	_, err = tokenSvc.AuthenticateSession(t.Context(), secondSession.Token)
	require.Error(t, err)
}
