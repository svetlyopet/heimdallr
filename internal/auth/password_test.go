package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
)

func TestIsLegacyPasswordHash(t *testing.T) {
	require.True(t, IsLegacyPasswordHash(legacyHashForTest("password")))
	require.False(t, IsLegacyPasswordHash("$2a$12$abcdefghijklmnopqrstuv"))
	require.False(t, IsLegacyPasswordHash("not-a-legacy-hash"))
	require.False(t, IsLegacyPasswordHash(strings.Repeat("g", 64)))
}

func TestServiceCountLegacyPasswordHashes(t *testing.T) {
	svc, _, db := newTestService(t, ServiceConfig{})

	count, err := svc.CountLegacyPasswordHashes(t.Context())
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	require.NoError(t, db.Create(&User{
		Username:              "legacy-one",
		Email:                 "legacy-one@example.com",
		PasswordHash:          legacyHashForTest("LegacyPassword123!"),
		PasswordResetRequired: true,
		Roles:                 []string{RoleReader},
	}).Error)

	_, err = svc.Create(t.Context(), api.AuthCreateUserRequest{
		Username: "modern-user",
		Email:    openapi_types.Email("modern-user@example.com"),
		Password: "StrongPassword123!",
	})
	require.NoError(t, err)

	count, err = svc.CountLegacyPasswordHashes(t.Context())
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestServiceUpdatePasswordClearsResetRequired(t *testing.T) {
	svc, repo, db := newTestService(t, ServiceConfig{})

	require.NoError(t, db.Create(&User{
		Username:              "reset-user",
		Email:                 "reset-user@example.com",
		PasswordHash:          legacyHashForTest("LegacyPassword123!"),
		PasswordResetRequired: true,
		Roles:                 []string{RoleReader},
	}).Error)

	created, err := repo.FindByUsername(t.Context(), "reset-user")
	require.NoError(t, err)

	newPassword := "StrongPasswordUpdated123!"
	_, err = svc.Update(t.Context(), created.ID.String(), api.AuthUpdateUserRequest{
		Password: &newPassword,
	})
	require.NoError(t, err)

	updated, err := repo.FindByID(t.Context(), created.ID.String())
	require.NoError(t, err)
	require.False(t, updated.PasswordResetRequired)
	require.True(t, strings.HasPrefix(updated.PasswordHash, "$2"))
}

func TestRepositoryCreateRejectsNonBcryptHash(t *testing.T) {
	_, repo, _ := newTestService(t, ServiceConfig{})

	_, err := repo.Create(t.Context(), User{
		Username:     "bad-hash-user",
		Email:        "bad-hash-user@example.com",
		PasswordHash: legacyHashForTest("LegacyPassword123!"),
		Roles:        []string{RoleReader},
	})
	require.ErrorIs(t, err, ErrInvalidPasswordValue)
}

func mustHashPassword(t *testing.T, value string) string {
	t.Helper()

	hash, err := hashPassword(value)
	require.NoError(t, err)
	return hash
}

func legacyHashForTest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
