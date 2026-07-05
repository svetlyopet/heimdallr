package token

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

func newTokenService(t *testing.T) Service {
	t.Helper()

	db := testutil.NewSQLiteDB(t, &APIToken{})
	return NewService(NewRepository(db), nil)
}

func TestServiceCreateAndAuthenticateToken(t *testing.T) {
	svc := newTokenService(t)

	created, err := svc.Create(context.Background(), CreateRequest{
		Name:   "ci-token",
		Scopes: []string{ScopeApplicationWrite},
	}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, created.Token)

	user, err := svc.Authenticate(context.Background(), created.Token)
	require.NoError(t, err)
	require.Equal(t, "token:ci-token", user.Username)
	require.True(t, svc.HasScope(user, ScopeApplicationWrite))
	require.False(t, svc.HasScope(user, ScopeAutomationWrite))
}

func TestServiceCreateReturnsInvalidScopes(t *testing.T) {
	svc := newTokenService(t)

	_, err := svc.Create(context.Background(), CreateRequest{
		Name:   "bad-token",
		Scopes: []string{"invalid:scope"},
	}, nil)
	require.ErrorIs(t, err, ErrInvalidScopes)
}

func TestServiceAuthenticateReturnsInvalidToken(t *testing.T) {
	svc := newTokenService(t)

	_, err := svc.Authenticate(context.Background(), "missing-token")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestServiceDeleteReturnsNotFound(t *testing.T) {
	svc := newTokenService(t)

	err := svc.Delete(context.Background(), "00000000-0000-0000-0000-000000000001")
	require.ErrorIs(t, err, ErrTokenNotFound)
}

func TestServiceHasScopeAllowsAdmin(t *testing.T) {
	svc := newTokenService(t)

	user := auth.GetResponse{Roles: []string{auth.RoleAdmin}}
	require.True(t, svc.HasScope(user, ScopeApplicationWrite))
}
