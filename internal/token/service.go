package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/svetlyopet/heimdallr/internal/auth"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/token/api"
	"gorm.io/gorm"
)

type Service interface {
	List(ctx context.Context) ([]api.Token, error)
	Create(ctx context.Context, req api.TokenCreateRequest, createdBy *uuid.UUID) (api.TokenCreateResponse, error)
	CreateSession(ctx context.Context, name string, scopes []string, createdBy uuid.UUID) (api.TokenCreateResponse, error)
	Delete(ctx context.Context, tokenID string) error
	Authenticate(ctx context.Context, plainToken string) (authapi.AuthUser, error)
	AuthenticateSession(ctx context.Context, plainToken string) (authapi.AuthUser, error)
	RevokeSessionTokens(ctx context.Context, userID string) error
	RevokeAllUserTokens(ctx context.Context, userID string) error
	RevokeSessionToken(ctx context.Context, plainToken string) error
}

type service struct {
	repository     Repository
	userRepository auth.Repository
	logger         *logger.Logger
	sessionTTL     time.Duration
	defaultAPITTL  time.Duration
	maxAPITTL      time.Duration
}

func (s service) List(ctx context.Context) ([]api.Token, error) {
	tokens, err := s.repository.FindAll(ctx)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to list api tokens", err)
		return nil, ErrListTokens
	}

	responses := make([]api.Token, 0, len(tokens))
	for _, token := range tokens {
		responses = append(responses, mapEntityToResponse(token))
	}

	return responses, nil
}

func (s service) Create(ctx context.Context, req api.TokenCreateRequest, createdBy *uuid.UUID) (api.TokenCreateResponse, error) {
	scopes := rbac.NormalizeScopes(scopesFromAPI(req.Scopes))
	if len(scopes) == 0 {
		return api.TokenCreateResponse{}, ErrInvalidScopes
	}

	ttl := s.defaultAPITTL
	if req.TtlSeconds != nil {
		ttl = time.Duration(*req.TtlSeconds) * time.Second
	}
	if ttl <= 0 || ttl > s.maxAPITTL {
		return api.TokenCreateResponse{}, ErrInvalidTTL
	}

	plainToken, tokenHash, err := generateToken()
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to generate api token", err)
		return api.TokenCreateResponse{}, ErrCreateToken
	}

	expiresAt := time.Now().UTC().Add(ttl)
	token := APIToken{
		ID:        uuid.New(),
		Name:      req.Name,
		TokenHash: tokenHash,
		Scopes:    scopes,
		Kind:      TokenKindAPI,
		ExpiresAt: &expiresAt,
		CreatedBy: createdBy,
	}

	created, err := s.repository.Create(ctx, token)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create api token", err, slog.String("name", req.Name))
		return api.TokenCreateResponse{}, ErrCreateToken
	}

	response := mapEntityToResponse(created)
	return api.TokenCreateResponse{
		CreatedBy: response.CreatedBy,
		Id:        response.Id,
		Name:      response.Name,
		Scopes:    scopesToAPI(scopes),
		ExpiresAt: response.ExpiresAt,
		Token:     plainToken,
	}, nil
}

func (s service) CreateSession(ctx context.Context, name string, scopes []string, createdBy uuid.UUID) (api.TokenCreateResponse, error) {
	normalizedScopes := rbac.NormalizeScopes(scopes)
	if len(normalizedScopes) == 0 {
		return api.TokenCreateResponse{}, ErrInvalidScopes
	}

	plainToken, tokenHash, err := generateToken()
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to generate session token", err)
		return api.TokenCreateResponse{}, ErrCreateToken
	}

	expiresAt := time.Now().UTC().Add(s.sessionTTL)
	token := APIToken{
		ID:        uuid.New(),
		Name:      name,
		TokenHash: tokenHash,
		Scopes:    normalizedScopes,
		Kind:      TokenKindSession,
		ExpiresAt: &expiresAt,
		CreatedBy: &createdBy,
	}

	created, err := s.repository.Create(ctx, token)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create session token", err, slog.String("name", name))
		return api.TokenCreateResponse{}, ErrCreateToken
	}

	response := mapEntityToResponse(created)
	return api.TokenCreateResponse{
		CreatedBy: response.CreatedBy,
		Id:        response.Id,
		Name:      response.Name,
		Scopes:    scopesToAPI(normalizedScopes),
		ExpiresAt: response.ExpiresAt,
		Token:     plainToken,
	}, nil
}

func (s service) Delete(ctx context.Context, tokenID string) error {
	if err := s.repository.DeleteByID(ctx, tokenID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTokenNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to delete api token", err, slog.String("token_id", tokenID))
		return ErrDeleteToken
	}

	return nil
}

func (s service) RevokeSessionTokens(ctx context.Context, userID string) error {
	if err := s.repository.DeleteSessionTokensByCreatedBy(ctx, userID); err != nil {
		s.logger.ErrorWithStack(ctx, "failed to revoke session tokens", err, slog.String("user_id", userID))
		return ErrDeleteToken
	}

	return nil
}

func (s service) RevokeAllUserTokens(ctx context.Context, userID string) error {
	if err := s.repository.DeleteByCreatedBy(ctx, userID); err != nil {
		s.logger.ErrorWithStack(ctx, "failed to revoke user tokens", err, slog.String("user_id", userID))
		return ErrDeleteToken
	}

	return nil
}

func (s service) RevokeSessionToken(ctx context.Context, plainToken string) error {
	if plainToken == "" {
		return nil
	}

	if err := s.repository.DeleteSessionByHash(ctx, hashToken(plainToken)); err != nil {
		s.logger.ErrorWithStack(ctx, "failed to revoke session token", err)
		return ErrDeleteToken
	}

	return nil
}

func (s service) Authenticate(ctx context.Context, plainToken string) (authapi.AuthUser, error) {
	return s.authenticate(ctx, plainToken, "")
}

func (s service) AuthenticateSession(ctx context.Context, plainToken string) (authapi.AuthUser, error) {
	return s.authenticate(ctx, plainToken, TokenKindSession)
}

func (s service) authenticate(ctx context.Context, plainToken string, requiredKind string) (authapi.AuthUser, error) {
	if plainToken == "" {
		return authapi.AuthUser{}, ErrInvalidToken
	}

	tokenHash := hashToken(plainToken)
	apiToken, err := s.repository.FindByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return authapi.AuthUser{}, ErrInvalidToken
		}

		s.logger.ErrorWithStack(ctx, "failed to authenticate api token", err)
		return authapi.AuthUser{}, ErrInvalidToken
	}

	if apiToken.ExpiresAt == nil || !apiToken.ExpiresAt.After(time.Now().UTC()) {
		return authapi.AuthUser{}, ErrInvalidToken
	}
	if requiredKind != "" && apiToken.Kind != requiredKind {
		return authapi.AuthUser{}, ErrInvalidToken
	}

	var roles []string
	userID := apiToken.ID.String()
	username := "token:" + apiToken.Name
	email := "token@heimdallr.local"

	if apiToken.CreatedBy != nil {
		userID = apiToken.CreatedBy.String()

		user, userErr := s.userRepository.FindByID(ctx, userID)
		if userErr != nil {
			if errors.Is(userErr, gorm.ErrRecordNotFound) {
				return authapi.AuthUser{}, ErrInvalidToken
			}

			s.logger.ErrorWithStack(ctx, "failed to load token owner", userErr, slog.String("user_id", userID))
			return authapi.AuthUser{}, ErrInvalidToken
		}

		username = user.Username
		email = user.Email

		if apiToken.Kind == TokenKindSession {
			roles = rbac.RolesFromLiveUser(user.Roles)
		}
	}

	if len(roles) == 0 {
		roles = rbac.ScopesToRoles(apiToken.Scopes)
	}

	authRoles := make([]authapi.AuthRole, 0, len(roles))
	for _, role := range roles {
		authRoles = append(authRoles, authapi.AuthRole(role))
	}

	return authapi.AuthUser{
		Id:       userID,
		Username: username,
		Email:    openapi_types.Email(email),
		Roles:    authRoles,
	}, nil
}

func NewService(repository Repository, userRepository auth.Repository, appLogger *logger.Logger, cfg ServiceConfig) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	defaults := DefaultServiceConfig()
	if cfg.SessionTokenTTL <= 0 {
		cfg.SessionTokenTTL = defaults.SessionTokenTTL
	}
	if cfg.DefaultAPITokenTTL <= 0 {
		cfg.DefaultAPITokenTTL = defaults.DefaultAPITokenTTL
	}
	if cfg.MaxAPITokenTTL <= 0 {
		cfg.MaxAPITokenTTL = defaults.MaxAPITokenTTL
	}
	if cfg.DefaultAPITokenTTL > cfg.MaxAPITokenTTL {
		cfg.DefaultAPITokenTTL = cfg.MaxAPITokenTTL
	}

	return &service{
		repository:     repository,
		userRepository: userRepository,
		logger:         appLogger,
		sessionTTL:     cfg.SessionTokenTTL,
		defaultAPITTL:  cfg.DefaultAPITokenTTL,
		maxAPITTL:      cfg.MaxAPITokenTTL,
	}
}

func mapEntityToResponse(token APIToken) api.Token {
	return api.Token{
		Id:        token.ID,
		Name:      token.Name,
		Scopes:    scopesToAPI(token.Scopes),
		CreatedBy: token.CreatedBy,
		ExpiresAt: token.ExpiresAt,
	}
}

func scopesFromAPI(scopes []api.TokenScope) []string {
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		result = append(result, string(scope))
	}

	return result
}

func scopesToAPI(scopes []string) []api.TokenScope {
	result := make([]api.TokenScope, 0, len(scopes))
	for _, scope := range scopes {
		result = append(result, api.TokenScope(scope))
	}

	return result
}

func generateToken() (string, string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", "", err
	}

	plainToken := hex.EncodeToString(buffer)
	return plainToken, hashToken(plainToken), nil
}

func hashToken(plainToken string) string {
	sum := sha256.Sum256([]byte(plainToken))
	return hex.EncodeToString(sum[:])
}
