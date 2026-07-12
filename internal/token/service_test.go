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
	"golang.org/x/crypto/bcrypt"
)

func newTokenService(t *testing.T) Service {
	t.Helper()

	db := testutil.NewPostgresDB(t)
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
	require.NotNil(t, created.ExpiresAt)
	require.WithinDuration(t, time.Now().UTC().Add(90*24*time.Hour), *created.ExpiresAt, time.Second)

	user, err := svc.Authenticate(context.Background(), created.Token)
	require.NoError(t, err)
	require.Equal(t, "token:ci-token", user.Username)

	_, err = svc.AuthenticateSession(context.Background(), created.Token)
	require.ErrorIs(t, err, ErrInvalidToken)

	authorizer := rbac.NewAuthorizer()
	require.True(t, authorizer.HasScope(user, rbac.ScopeApplicationWrite))
	require.False(t, authorizer.HasScope(user, rbac.ScopeAutomationWrite))
}

func TestServiceCreateHonorsBoundedTTL(t *testing.T) {
	svc := newTokenService(t)
	ttlSeconds := 60

	created, err := svc.Create(context.Background(), api.TokenCreateRequest{
		Name:       "short-lived",
		Scopes:     []api.TokenScope{api.Read},
		TtlSeconds: &ttlSeconds,
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, created.ExpiresAt)
	require.WithinDuration(t, time.Now().UTC().Add(time.Minute), *created.ExpiresAt, time.Second)

	tooLong := 365*24*60*60 + 1
	_, err = svc.Create(context.Background(), api.TokenCreateRequest{
		Name:       "too-long",
		Scopes:     []api.TokenScope{api.Read},
		TtlSeconds: &tooLong,
	}, nil)
	require.ErrorIs(t, err, ErrInvalidTTL)
}

func TestServiceCreateSessionUsesLiveUserRoles(t *testing.T) {
	svc := newTokenService(t).(*service)
	db := testutil.NewPostgresDB(t)
	userRepo := auth.NewRepository(db)
	svc.userRepository = userRepo
	svc.repository = NewRepository(db)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("StrongPassword123!"), bcrypt.DefaultCost)
	require.NoError(t, err)

	createdUser, err := userRepo.Create(context.Background(), auth.User{
		Username:     "reader-user",
		Email:        "reader@example.com",
		PasswordHash: string(passwordHash),
		Roles:        []string{auth.RoleReader},
	})
	require.NoError(t, err)

	session, err := svc.CreateSession(context.Background(), "session-reader-user", []string{rbac.ScopeAdmin}, createdUser.ID)
	require.NoError(t, err)

	user, err := svc.AuthenticateSession(context.Background(), session.Token)
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

func TestDatabaseRejectsTokenWithoutExpiration(t *testing.T) {
	svc := newTokenService(t).(*service)
	_, hash, err := generateToken()
	require.NoError(t, err)

	err = svc.repository.(*repository).db.Exec(
		`INSERT INTO api_tokens (id, name, token_hash, scopes, kind, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.New(), "legacy", hash, `["read"]`, TokenKindAPI, time.Now().UTC(), time.Now().UTC(),
	).Error
	require.Error(t, err)
}

func TestServiceRevokeSessionTokenDoesNotRevokeAPIToken(t *testing.T) {
	svc := newTokenService(t).(*service)
	apiToken, err := svc.Create(context.Background(), api.TokenCreateRequest{
		Name:   "api-token",
		Scopes: []api.TokenScope{api.Read},
	}, nil)
	require.NoError(t, err)

	require.NoError(t, svc.RevokeSessionToken(context.Background(), apiToken.Token))
	_, err = svc.Authenticate(context.Background(), apiToken.Token)
	require.NoError(t, err)
}

func TestServiceDeleteReturnsNotFound(t *testing.T) {
	svc := newTokenService(t)

	err := svc.Delete(context.Background(), "00000000-0000-0000-0000-000000000001")
	require.ErrorIs(t, err, ErrTokenNotFound)
}
