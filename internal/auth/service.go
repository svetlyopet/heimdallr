package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"gorm.io/gorm"
)

const (
	rootUsername        = "root"
	rootDefaultEmail    = "root@localhost"
	rootPasswordLength  = 24
	minimumPasswordSize = 12
)

type Service interface {
	Authenticate(ctx context.Context, username string, password string) (api.AuthUser, error)
	List(ctx context.Context) ([]api.AuthUser, error)
	Create(ctx context.Context, req api.AuthCreateUserRequest) (api.AuthUser, error)
	Update(ctx context.Context, userID string, req api.AuthUpdateUserRequest) (api.AuthUser, error)
	Delete(ctx context.Context, userID string) error
	EnsureRootUser(ctx context.Context) (string, error)
	CountLegacyPasswordHashes(ctx context.Context) (int64, error)
	HasAnyRole(user api.AuthUser, requiredRoles ...string) bool
}

type ServiceConfig struct {
	BootstrapRootPassword string
}

type service struct {
	repository        Repository
	tokenRepo         TokenRepository
	db                *gorm.DB
	logger            *logger.Logger
	bootstrapPassword string
	supportedRoles    map[string]struct{}
}

func (s service) Authenticate(ctx context.Context, username string, password string) (api.AuthUser, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return api.AuthUser{}, ErrInvalidCredentials
	}

	user, err := s.repository.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.AuthUser{}, ErrInvalidCredentials
		}

		s.logger.ErrorWithStack(ctx, "failed to authenticate user", err, slog.String("username", username))
		return api.AuthUser{}, ErrInvalidCredentials
	}

	if user.PasswordResetRequired {
		return api.AuthUser{}, ErrInvalidCredentials
	}

	valid, _ := verifyPassword(password, user.PasswordHash)
	if !valid {
		return api.AuthUser{}, ErrInvalidCredentials
	}

	return mapEntityToResponse(user), nil
}

func (s service) Create(ctx context.Context, req api.AuthCreateUserRequest) (api.AuthUser, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(string(req.Email))

	if username == "" || email == "" {
		return api.AuthUser{}, ErrInvalidCredentials
	}

	if len(req.Password) < minimumPasswordSize || strings.TrimSpace(req.Password) == "" {
		return api.AuthUser{}, ErrInvalidPasswordValue
	}

	roles, err := s.validateRoles(rolesFromAPI(req.Roles), true)
	if err != nil {
		return api.AuthUser{}, err
	}

	_, err = s.repository.FindByUsername(ctx, username)
	if err == nil {
		return api.AuthUser{}, ErrUserAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.ErrorWithStack(ctx, "failed to check user existence before create", err, slog.String("username", username))
		return api.AuthUser{}, ErrCreateUser
	}

	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to hash password", err, slog.String("username", username))
		return api.AuthUser{}, ErrCreateUser
	}

	created, err := s.repository.Create(ctx, User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Roles:        roles,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return api.AuthUser{}, ErrUserAlreadyExists
		}

		s.logger.ErrorWithStack(ctx, "failed to create user", err, slog.String("username", username))
		return api.AuthUser{}, ErrCreateUser
	}

	return mapEntityToResponse(created), nil
}

func (s service) List(ctx context.Context) ([]api.AuthUser, error) {
	users, err := s.repository.List(ctx)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to list users", err)
		return nil, ErrListUsers
	}

	responses := make([]api.AuthUser, 0, len(users))
	for _, user := range users {
		responses = append(responses, mapEntityToResponse(user))
	}

	return responses, nil
}

func (s service) Update(ctx context.Context, userID string, req api.AuthUpdateUserRequest) (api.AuthUser, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return api.AuthUser{}, ErrInvalidUserID
	}

	existing, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.AuthUser{}, ErrUserNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find user for update", err, slog.String("user_id", userID))
		return api.AuthUser{}, ErrUpdateUser
	}

	update := User{}
	revokeSessions := false

	if req.Email != nil {
		email := strings.TrimSpace(string(*req.Email))
		if email == "" {
			return api.AuthUser{}, ErrInvalidCredentials
		}
		update.Email = email
	}

	if req.Password != nil {
		if len(*req.Password) < minimumPasswordSize || strings.TrimSpace(*req.Password) == "" {
			return api.AuthUser{}, ErrInvalidPasswordValue
		}

		passwordHash, hashErr := hashPassword(*req.Password)
		if hashErr != nil {
			s.logger.ErrorWithStack(ctx, "failed to hash password", hashErr, slog.String("user_id", userID))
			return api.AuthUser{}, ErrUpdateUser
		}

		update.PasswordHash = passwordHash
		revokeSessions = true
	}

	if req.Roles != nil {
		roles, roleErr := s.validateRoles(rolesFromAPI(req.Roles), false)
		if roleErr != nil {
			return api.AuthUser{}, roleErr
		}
		if len(roles) == 0 {
			roles = existing.Roles
		}

		if existing.Username == rootUsername && !slices.Equal(roles, existing.Roles) {
			return api.AuthUser{}, ErrRootRoleForbidden
		}

		if !slices.Equal(roles, existing.Roles) {
			revokeSessions = true
		}

		update.Roles = roles
	}

	var updated User

	if revokeSessions {
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var updateErr error
			updated, updateErr = s.repository.WithTx(tx).UpdateByID(ctx, userID, update)
			if updateErr != nil {
				return updateErr
			}

			return s.tokenRepo.WithTx(tx).DeleteSessionTokensByCreatedBy(ctx, userID)
		})
	} else {
		updated, err = s.repository.UpdateByID(ctx, userID, update)
	}

	if err != nil {
		if errors.Is(err, ErrConcurrentUserUpdate) {
			return api.AuthUser{}, ErrConcurrentUserUpdate
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.AuthUser{}, ErrUserNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to update user", err, slog.String("user_id", userID))
		return api.AuthUser{}, ErrUpdateUser
	}

	return mapEntityToResponse(updated), nil
}

func (s service) Delete(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrInvalidUserID
	}

	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to fetch user before delete", err, slog.String("user_id", userID))
		return ErrDeleteUser
	}

	if user.Username == rootUsername {
		return ErrRootDeleteForbidden
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.tokenRepo.WithTx(tx).DeleteByCreatedBy(ctx, userID); err != nil {
			return err
		}

		return s.repository.WithTx(tx).DeleteByID(ctx, userID)
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to delete user", err, slog.String("user_id", userID))
		return ErrDeleteUser
	}

	return nil
}

func (s service) EnsureRootUser(ctx context.Context) (string, error) {
	_, err := s.repository.FindByUsername(ctx, rootUsername)
	if err == nil {
		return "", nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.ErrorWithStack(ctx, "failed to check root user existence", err)
		return "", ErrRootBootstrap
	}

	password := strings.TrimSpace(s.bootstrapPassword)
	if password == "" {
		var genErr error
		password, genErr = generateSecurePassword(rootPasswordLength)
		if genErr != nil {
			s.logger.ErrorWithStack(ctx, "failed to generate root password", genErr)
			return "", ErrRootBootstrap
		}
	} else if len(password) < minimumPasswordSize {
		return "", ErrRootBootstrap
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to hash root password", err)
		return "", ErrRootBootstrap
	}

	_, err = s.repository.Create(ctx, User{
		Username:     rootUsername,
		Email:        rootDefaultEmail,
		PasswordHash: passwordHash,
		Roles:        []string{RoleAdmin},
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return "", nil
		}

		s.logger.ErrorWithStack(ctx, "failed to create root user", err)
		return "", ErrRootBootstrap
	}

	return password, nil
}

func (s service) CountLegacyPasswordHashes(ctx context.Context) (int64, error) {
	count, err := s.repository.CountLegacyPasswordHashes(ctx)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to count legacy password hashes", err)
		return 0, ErrListUsers
	}

	return count, nil
}

func (s service) HasAnyRole(user api.AuthUser, requiredRoles ...string) bool {
	if len(requiredRoles) == 0 {
		return false
	}

	userRoles := map[string]struct{}{}
	for _, role := range normalizeRoles(rolesFromSlice(user.Roles)) {
		userRoles[role] = struct{}{}
	}

	for _, role := range normalizeRoles(requiredRoles) {
		if _, ok := userRoles[role]; ok {
			return true
		}
	}

	return false
}

func (s service) validateRoles(roles []string, applyDefault bool) ([]string, error) {
	normalized := normalizeRoles(roles)
	if len(normalized) == 0 && applyDefault {
		normalized = []string{RoleReader}
	}

	for _, role := range normalized {
		if _, ok := s.supportedRoles[role]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrInvalidRole, role)
		}
	}

	return normalized, nil
}

func NewService(repository Repository, tokenRepo TokenRepository, db *gorm.DB, appLogger *logger.Logger, cfg ServiceConfig) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	roles := map[string]struct{}{}
	for role := range defaultSupportedRoles {
		roles[role] = struct{}{}
	}

	return &service{
		repository:        repository,
		tokenRepo:         tokenRepo,
		db:                db,
		logger:            appLogger,
		bootstrapPassword: strings.TrimSpace(cfg.BootstrapRootPassword),
		supportedRoles:    roles,
	}
}

func mapEntityToResponse(user User) api.AuthUser {
	return api.AuthUser{
		Id:       user.ID.String(),
		Username: user.Username,
		Email:    emailToAPI(user.Email),
		Roles:    rolesToAPI(user.Roles),
	}
}

// LoginScopesForRoles maps user roles to token scopes at login time.
func LoginScopesForRoles(roles []string) []string {
	return rbac.LoginScopesForRoles(roles)
}
