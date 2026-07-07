package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
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

	created, err := svc.Create(t.Context(), CreateRequest{
		Username: "reader-user",
		Email:    "reader-user@example.com",
		Password: "StrongPassword123!",
	})
	require.NoError(t, err)
	require.Equal(t, []string{RoleReader}, created.Roles)

	_, err = svc.Create(t.Context(), CreateRequest{
		Username: "bad-role-user",
		Email:    "bad-role-user@example.com",
		Password: "StrongPassword123!",
		Roles:    []string{"owner"},
	})
	require.ErrorIs(t, err, ErrInvalidRole)

	_, err = svc.Create(t.Context(), CreateRequest{
		Username: "   ",
		Email:    "blank@example.com",
		Password: "StrongPassword123!",
	})
	require.ErrorIs(t, err, ErrInvalidCredentials)

	_, err = svc.Create(t.Context(), CreateRequest{
		Username: "short-password",
		Email:    "short-password@example.com",
		Password: "short",
	})
	require.ErrorIs(t, err, ErrInvalidPasswordValue)
}

func TestServiceCreateDuplicateUsername(t *testing.T) {
	svc, _, _ := newTestService(t, ServiceConfig{})

	_, err := svc.Create(t.Context(), CreateRequest{
		Username: "duplicate-user",
		Email:    "first@example.com",
		Password: "StrongPassword123!",
		Roles:    []string{RoleReader},
	})
	require.NoError(t, err)

	_, err = svc.Create(t.Context(), CreateRequest{
		Username: "duplicate-user",
		Email:    "second@example.com",
		Password: "StrongPassword123!",
		Roles:    []string{RoleReader},
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

	created, err := svc.Create(t.Context(), CreateRequest{
		Username: "managed-user",
		Email:    "managed-user@example.com",
		Password: "StrongPassword123!",
		Roles:    []string{RoleReader},
	})
	require.NoError(t, err)

	listed, err := svc.List(t.Context())
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, created.Username, listed[0].Username)

	updated, err := svc.Update(t.Context(), created.ID, UpdateRequest{
		Email:    "managed-user-updated@example.com",
		Password: "StrongPasswordUpdated123!",
		Roles:    []string{RoleAdmin},
	})
	require.NoError(t, err)
	require.Equal(t, "managed-user-updated@example.com", updated.Email)
	require.Equal(t, []string{RoleAdmin}, updated.Roles)

	_, err = svc.Update(t.Context(), created.ID, UpdateRequest{Roles: []string{"owner"}})
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

	_, err = svc.Update(t.Context(), root.ID.String(), UpdateRequest{Roles: []string{RoleReader}})
	require.ErrorIs(t, err, ErrRootRoleForbidden)

	updatedRoot, err := repo.FindByID(t.Context(), root.ID.String())
	require.NoError(t, err)
	require.Equal(t, []string{RoleAdmin}, updatedRoot.Roles)
}
