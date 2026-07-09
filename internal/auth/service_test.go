package auth

import (
	"fmt"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestService(t *testing.T, cfg ServiceConfig) (Service, Repository, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&User{}))

	repo := NewRepository(db)
	svc := NewService(repo, nil, cfg)

	return svc, repo, db
}

func TestServiceEnsureRootUserUsesBootstrapPassword(t *testing.T) {
	svc, _, _ := newTestService(t, ServiceConfig{BootstrapRootPassword: "EnvBootstrapPassword12!"})

	password, err := svc.EnsureRootUser(t.Context())
	require.NoError(t, err)
	require.Equal(t, "EnvBootstrapPassword12!", password)
}

func TestServiceEnsureRootUserCreatesOnce(t *testing.T) {
	svc, repo, _ := newTestService(t, ServiceConfig{})

	password, err := svc.EnsureRootUser(t.Context())
	require.NoError(t, err)
	require.Len(t, password, rootPasswordLength)

	root, err := repo.FindByUsername(t.Context(), rootUsername)
	require.NoError(t, err)
	require.Equal(t, rootUsername, root.Username)
	require.Equal(t, rootDefaultEmail, root.Email)
	require.Equal(t, []string{RoleAdmin}, root.Roles)
	require.NotEqual(t, password, root.PasswordHash)
	require.Equal(t, hashPassword(password), root.PasswordHash)

	password, err = svc.EnsureRootUser(t.Context())
	require.NoError(t, err)
	require.Empty(t, password)
}

func TestServiceCreateDefaultRoleAndValidation(t *testing.T) {
	svc, _, _ := newTestService(t, ServiceConfig{})

	created, err := svc.Create(t.Context(), api.AuthCreateUserRequest{
		Username: "reader-user",
		Email:    openapi_types.Email("reader-user@example.com"),
		Password: "StrongPassword123!",
	})
	require.NoError(t, err)
	require.Equal(t, []api.AuthRole{api.Reader}, created.Roles)

	owner := api.AuthRole("owner")
	_, err = svc.Create(t.Context(), api.AuthCreateUserRequest{
		Username: "bad-role-user",
		Email:    openapi_types.Email("bad-role-user@example.com"),
		Password: "StrongPassword123!",
		Roles:    &[]api.AuthRole{owner},
	})
	require.ErrorIs(t, err, ErrInvalidRole)

	_, err = svc.Create(t.Context(), api.AuthCreateUserRequest{
		Username: "   ",
		Email:    openapi_types.Email("blank@example.com"),
		Password: "StrongPassword123!",
	})
	require.ErrorIs(t, err, ErrInvalidCredentials)

	_, err = svc.Create(t.Context(), api.AuthCreateUserRequest{
		Username: "short-password",
		Email:    openapi_types.Email("short-password@example.com"),
		Password: "short",
	})
	require.ErrorIs(t, err, ErrInvalidPasswordValue)
}

func TestServiceCreateDuplicateUsername(t *testing.T) {
	svc, _, _ := newTestService(t, ServiceConfig{})

	_, err := svc.Create(t.Context(), api.AuthCreateUserRequest{
		Username: "duplicate-user",
		Email:    openapi_types.Email("first@example.com"),
		Password: "StrongPassword123!",
		Roles:    &[]api.AuthRole{api.Reader},
	})
	require.NoError(t, err)

	_, err = svc.Create(t.Context(), api.AuthCreateUserRequest{
		Username: "duplicate-user",
		Email:    openapi_types.Email("second@example.com"),
		Password: "StrongPassword123!",
		Roles:    &[]api.AuthRole{api.Reader},
	})
	require.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestServiceDeleteErrors(t *testing.T) {
	svc, _, _ := newTestService(t, ServiceConfig{})

	err := svc.Delete(t.Context(), "   ")
	require.ErrorIs(t, err, ErrInvalidUserID)

	err = svc.Delete(t.Context(), "5d8dd803-fca6-4f7c-9dd2-24417622d630")
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestServiceDeleteRejectsRootUser(t *testing.T) {
	svc, repo, _ := newTestService(t, ServiceConfig{})

	password, err := svc.EnsureRootUser(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, password)

	root, err := repo.FindByUsername(t.Context(), rootUsername)
	require.NoError(t, err)

	err = svc.Delete(t.Context(), root.ID.String())
	require.ErrorIs(t, err, ErrRootDeleteForbidden)

	_, err = repo.FindByID(t.Context(), root.ID.String())
	require.NoError(t, err)
}

func TestServiceListAndUpdate(t *testing.T) {
	svc, _, _ := newTestService(t, ServiceConfig{})

	created, err := svc.Create(t.Context(), api.AuthCreateUserRequest{
		Username: "managed-user",
		Email:    openapi_types.Email("managed-user@example.com"),
		Password: "StrongPassword123!",
		Roles:    &[]api.AuthRole{api.Reader},
	})
	require.NoError(t, err)

	listed, err := svc.List(t.Context())
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, created.Username, listed[0].Username)

	updatedEmail := openapi_types.Email("managed-user-updated@example.com")
	updatedPassword := "StrongPasswordUpdated123!"
	updatedRoles := []api.AuthRole{api.Admin}
	updated, err := svc.Update(t.Context(), created.Id, api.AuthUpdateUserRequest{
		Email:    &updatedEmail,
		Password: &updatedPassword,
		Roles:    &updatedRoles,
	})
	require.NoError(t, err)
	require.Equal(t, updatedEmail, updated.Email)
	require.Equal(t, []api.AuthRole{api.Admin}, updated.Roles)

	owner := api.AuthRole("owner")
	_, err = svc.Update(t.Context(), created.Id, api.AuthUpdateUserRequest{Roles: &[]api.AuthRole{owner}})
	require.ErrorIs(t, err, ErrInvalidRole)
}

func TestServiceUpdateRejectsRootRoleChange(t *testing.T) {
	svc, repo, _ := newTestService(t, ServiceConfig{})

	password, err := svc.EnsureRootUser(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, password)

	root, err := repo.FindByUsername(t.Context(), rootUsername)
	require.NoError(t, err)
	require.Equal(t, []string{RoleAdmin}, root.Roles)

	readerRoles := []api.AuthRole{api.Reader}
	_, err = svc.Update(t.Context(), root.ID.String(), api.AuthUpdateUserRequest{Roles: &readerRoles})
	require.ErrorIs(t, err, ErrRootRoleForbidden)

	updatedRoot, err := repo.FindByID(t.Context(), root.ID.String())
	require.NoError(t, err)
	require.Equal(t, []string{RoleAdmin}, updatedRoot.Roles)
}
