package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/auth"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/token/api"
	"gorm.io/gorm"
)

type Service interface {
	List(ctx context.Context) ([]api.Token, error)
	Create(ctx context.Context, req api.TokenCreateRequest, createdBy *uuid.UUID) (api.TokenCreateResponse, error)
	Delete(ctx context.Context, tokenID string) error
	Authenticate(ctx context.Context, plainToken string) (authapi.AuthUser, error)
	HasScope(user authapi.AuthUser, scope string) bool
}

type service struct {
	repository Repository
	logger     *logger.Logger
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
	scopes := normalizeScopes(scopesFromAPI(req.Scopes))
	if len(scopes) == 0 {
		return api.TokenCreateResponse{}, ErrInvalidScopes
	}

	plainToken, tokenHash, err := generateToken()
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to generate api token", err)
		return api.TokenCreateResponse{}, ErrCreateToken
	}

	token := APIToken{
		ID:        uuid.New(),
		Name:      req.Name,
		TokenHash: tokenHash,
		Scopes:    scopes,
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

func (s service) Authenticate(ctx context.Context, plainToken string) (authapi.AuthUser, error) {
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

	roles := scopesToRoles(apiToken.Scopes)
	authRoles := make([]authapi.AuthRole, 0, len(roles))
	for _, role := range roles {
		authRoles = append(authRoles, authapi.AuthRole(role))
	}

	userID := apiToken.ID.String()
	if apiToken.CreatedBy != nil {
		userID = apiToken.CreatedBy.String()
	}

	return authapi.AuthUser{
		Id:       userID,
		Username: "token:" + apiToken.Name,
		Email:    "token@heimdallr.local",
		Roles:    authRoles,
	}, nil
}

func (s service) HasScope(user authapi.AuthUser, scope string) bool {
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, string(role))
	}

	if slices.Contains(roles, auth.RoleAdmin) {
		return true
	}

	return slices.Contains(roles, scope)
}

func NewService(repository Repository, appLogger *logger.Logger) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &service{
		repository: repository,
		logger:     appLogger,
	}
}

func mapEntityToResponse(token APIToken) api.Token {
	return api.Token{
		Id:        token.ID,
		Name:      token.Name,
		Scopes:    scopesToAPI(token.Scopes),
		CreatedBy: token.CreatedBy,
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

func normalizeScopes(scopes []string) []string {
	allowed := map[string]struct{}{
		ScopeApplicationWrite: {},
		ScopeAutomationWrite:  {},
		ScopeRead:             {},
		ScopeAdmin:            {},
	}

	normalized := make([]string, 0, len(scopes))
	seen := map[string]struct{}{}

	for _, scope := range scopes {
		if _, ok := allowed[scope]; !ok {
			continue
		}

		if _, ok := seen[scope]; ok {
			continue
		}

		seen[scope] = struct{}{}
		normalized = append(normalized, scope)
	}

	return normalized
}

func scopesToRoles(scopes []string) []string {
	if slices.Contains(scopes, ScopeAdmin) {
		return []string{auth.RoleAdmin, auth.RoleReader}
	}

	roles := []string{auth.RoleReader}
	for _, scope := range scopes {
		if scope == ScopeApplicationWrite || scope == ScopeAutomationWrite || scope == ScopeRead {
			if !slices.Contains(roles, scope) {
				roles = append(roles, scope)
			}
		}
	}

	return roles
}
