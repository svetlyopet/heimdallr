package token

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"github.com/svetlyopet/heimdallr/internal/token/api"
)

func newTokenService(t *testing.T) Service {
	t.Helper()

	db := testutil.NewSQLiteDB(t, &APIToken{}, &auth.User{})
	return NewService(NewRepository(db), auth.NewRepository(db), nil, DefaultServiceConfig())
}

func TestServiceCreateAndAuthenticateToken(t *testing.T) {
	svc := newTokenService(t)

	created, err := svc.Create(context.Background(), api.TokenCreateRequest{
		Name:   "ci-token",
		Scopes: []api.TokenScope{api.ApplicationWrite},
	}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, created.Token)

	user, err := svc.Authenticate(context.Background(), created.Token)
	require.NoError(t, err)
	require.Equal(t, "token:ci-token", user.Username)

	authorizer := rbac.NewAuthorizer()
	require.True(t, authorizer.HasScope(user, rbac.ScopeApplicationWrite))
	require.False(t, authorizer.HasScope(user, rbac.ScopeAutomationWrite))
}

func TestServiceCreateSessionUsesLiveUserRoles(t *testing.T) {
	svc := newTokenService(t).(*service)
	db := testutil.NewSQLiteDB(t, &APIToken{}, &auth.User{})
	userRepo := auth.NewRepository(db)
	svc.userRepository = userRepo
	svc.repository = NewRepository(db)

	createdUser, err := userRepo.Create(context.Background(), auth.User{
		Username:     "reader-user",
		Email:        "reader@example.com",
		PasswordHash: "hash",
		Roles:        []string{auth.RoleReader},
	})
	require.NoError(t, err)

	session, err := svc.CreateSession(context.Background(), "session-reader-user", []string{rbac.ScopeAdmin}, createdUser.ID)
	require.NoError(t, err)

	user, err := svc.Authenticate(context.Background(), session.Token)
	require.NoError(t, err)
	require.Equal(t, createdUser.Username, user.Username)

	authorizer := rbac.NewAuthorizer()
	require.False(t, authorizer.HasScope(user, rbac.ScopeApplicationWrite))
}

func TestServiceCreateReturnsInvalidScopes(t *testing.T) {
	svc := newTokenService(t)

	_, err := svc.Create(context.Background(), api.TokenCreateRequest{
		Name:   "bad-token",
		Scopes: []api.TokenScope{"invalid:scope"},
	}, nil)
	require.ErrorIs(t, err, ErrInvalidScopes)
}

func TestServiceAuthenticateReturnsInvalidToken(t *testing.T) {
	svc := newTokenService(t)

	_, err := svc.Authenticate(context.Background(), "missing-token")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestServiceAuthenticateRejectsExpiredToken(t *testing.T) {
	svc := newTokenService(t).(*service)
	expired := time.Now().UTC().Add(-time.Hour)
	plain, hash, err := generateToken()
	require.NoError(t, err)

	_, err = svc.repository.Create(context.Background(), APIToken{
		ID:        uuid.New(),
		Name:      "expired",
		TokenHash: hash,
		Scopes:    []string{rbac.ScopeRead},
		Kind:      TokenKindAPI,
		ExpiresAt: &expired,
	})
	require.NoError(t, err)

	_, err = svc.Authenticate(context.Background(), plain)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestServiceDeleteReturnsNotFound(t *testing.T) {
	svc := newTokenService(t)

	err := svc.Delete(context.Background(), "00000000-0000-0000-0000-000000000001")
	require.ErrorIs(t, err, ErrTokenNotFound)
}
