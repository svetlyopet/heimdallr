package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/svetlyopet/heimdallr/internal/logger"
	"gorm.io/gorm"
)

const (
	rootUsername        = "root"
	rootDefaultEmail    = "root@localhost"
	rootPasswordLength  = 24
	minimumPasswordSize = 12
)

const passwordAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}<>?"

type Service interface {
	Authenticate(ctx context.Context, username string, password string) (GetResponse, error)
	List(ctx context.Context) ([]GetResponse, error)
	Create(ctx context.Context, req CreateRequest) (GetResponse, error)
	Update(ctx context.Context, userID string, req UpdateRequest) (GetResponse, error)
	Delete(ctx context.Context, userID string) error
	EnsureRootUser(ctx context.Context) (string, error)
	HasAnyRole(user GetResponse, requiredRoles ...string) bool
}

type service struct {
	repository     Repository
	logger         *logger.Logger
	supportedRoles map[string]struct{}
}

func (s service) Authenticate(ctx context.Context, username string, password string) (GetResponse, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return GetResponse{}, ErrInvalidCredentials
	}

	user, err := s.repository.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrInvalidCredentials
		}

		s.logger.ErrorWithStack(ctx, "failed to authenticate user", err, slog.String("username", username))
		return GetResponse{}, ErrInvalidCredentials
	}

	hashed := hashPassword(password)
	if subtle.ConstantTimeCompare([]byte(user.PasswordHash), []byte(hashed)) != 1 {
		return GetResponse{}, ErrInvalidCredentials
	}

	return mapEntityToResponse(user), nil
}

func (s service) Create(ctx context.Context, req CreateRequest) (GetResponse, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)

	if username == "" || email == "" {
		return GetResponse{}, ErrInvalidCredentials
	}

	if len(req.Password) < minimumPasswordSize || strings.TrimSpace(req.Password) == "" {
		return GetResponse{}, ErrInvalidPasswordValue
	}

	roles, err := s.validateRoles(req.Roles, true)
	if err != nil {
		return GetResponse{}, err
	}

	_, err = s.repository.FindByUsername(ctx, username)
	if err == nil {
		return GetResponse{}, ErrUserAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.ErrorWithStack(ctx, "failed to check user existence before create", err, slog.String("username", username))
		return GetResponse{}, ErrCreateUser
	}

	created, err := s.repository.Create(ctx, User{
		Username:     username,
		Email:        email,
		PasswordHash: hashPassword(req.Password),
		Roles:        roles,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return GetResponse{}, ErrUserAlreadyExists
		}

		s.logger.ErrorWithStack(ctx, "failed to create user", err, slog.String("username", username))
		return GetResponse{}, ErrCreateUser
	}

	return mapEntityToResponse(created), nil
}

func (s service) List(ctx context.Context) ([]GetResponse, error) {
	users, err := s.repository.List(ctx)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to list users", err)
		return nil, ErrListUsers
	}

	responses := make([]GetResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, mapEntityToResponse(user))
	}

	return responses, nil
}

func (s service) Update(ctx context.Context, userID string, req UpdateRequest) (GetResponse, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return GetResponse{}, ErrInvalidUserID
	}

	existing, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrUserNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find user for update", err, slog.String("user_id", userID))
		return GetResponse{}, ErrUpdateUser
	}

	update := User{}

	if req.Email != "" {
		email := strings.TrimSpace(req.Email)
		if email == "" {
			return GetResponse{}, ErrInvalidCredentials
		}
		update.Email = email
	}

	if req.Password != "" {
		if len(req.Password) < minimumPasswordSize || strings.TrimSpace(req.Password) == "" {
			return GetResponse{}, ErrInvalidPasswordValue
		}
		update.PasswordHash = hashPassword(req.Password)
	}

	if req.Roles != nil {
		roles, err := s.validateRoles(req.Roles, false)
		if err != nil {
			return GetResponse{}, err
		}
		if len(roles) == 0 {
			roles = existing.Roles
		}

		if existing.Username == rootUsername && !slices.Equal(roles, existing.Roles) {
			return GetResponse{}, ErrRootRoleForbidden
		}
		update.Roles = roles
	}

	updated, err := s.repository.UpdateByID(ctx, userID, update)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrUserNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to update user", err, slog.String("user_id", userID))
		return GetResponse{}, ErrUpdateUser
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

	if err := s.repository.DeleteByID(ctx, userID); err != nil {
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

	password, err := generateSecurePassword(rootPasswordLength)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to generate root password", err)
		return "", ErrRootBootstrap
	}

	_, err = s.repository.Create(ctx, User{
		Username:     rootUsername,
		Email:        rootDefaultEmail,
		PasswordHash: hashPassword(password),
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

func (s service) HasAnyRole(user GetResponse, requiredRoles ...string) bool {
	if len(requiredRoles) == 0 {
		return true
	}

	userRoles := map[string]struct{}{}
	for _, role := range normalizeRoles(user.Roles) {
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

func NewService(repository Repository, appLogger *logger.Logger) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	roles := map[string]struct{}{}
	for role := range defaultSupportedRoles {
		roles[role] = struct{}{}
	}

	return &service{repository: repository, logger: appLogger, supportedRoles: roles}
}

func hashPassword(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func generateSecurePassword(length int) (string, error) {
	if length <= 0 {
		return "", ErrInvalidPasswordValue
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i := range length {
		bytes[i] = passwordAlphabet[int(bytes[i])%len(passwordAlphabet)]
	}

	return string(bytes), nil
}

func mapEntityToResponse(user User) GetResponse {
	return GetResponse{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Roles:    user.Roles,
	}
}
