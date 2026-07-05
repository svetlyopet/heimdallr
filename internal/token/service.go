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
	"github.com/svetlyopet/heimdallr/internal/logger"
	"gorm.io/gorm"
)

type Service interface {
	List(ctx context.Context) ([]GetResponse, error)
	Create(ctx context.Context, req CreateRequest, createdBy *uuid.UUID) (CreateResponse, error)
	Delete(ctx context.Context, tokenID string) error
	Authenticate(ctx context.Context, plainToken string) (auth.GetResponse, error)
	HasScope(user auth.GetResponse, scope string) bool
}

type service struct {
	repository Repository
	logger     *logger.Logger
}

func (s service) List(ctx context.Context) ([]GetResponse, error) {
	tokens, err := s.repository.FindAll(ctx)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to list api tokens", err)
		return nil, ErrListTokens
	}

	responses := make([]GetResponse, 0, len(tokens))
	for _, token := range tokens {
		responses = append(responses, mapEntityToResponse(token))
	}

	return responses, nil
}

func (s service) Create(ctx context.Context, req CreateRequest, createdBy *uuid.UUID) (CreateResponse, error) {
	scopes := normalizeScopes(req.Scopes)
	if len(scopes) == 0 {
		return CreateResponse{}, ErrInvalidScopes
	}

	plainToken, tokenHash, err := generateToken()
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to generate api token", err)
		return CreateResponse{}, ErrCreateToken
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
		return CreateResponse{}, ErrCreateToken
	}

	return CreateResponse{
		GetResponse: mapEntityToResponse(created),
		Token:       plainToken,
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

func (s service) Authenticate(ctx context.Context, plainToken string) (auth.GetResponse, error) {
	if plainToken == "" {
		return auth.GetResponse{}, ErrInvalidToken
	}

	tokenHash := hashToken(plainToken)
	apiToken, err := s.repository.FindByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return auth.GetResponse{}, ErrInvalidToken
		}

		s.logger.ErrorWithStack(ctx, "failed to authenticate api token", err)
		return auth.GetResponse{}, ErrInvalidToken
	}

	roles := scopesToRoles(apiToken.Scopes)

	return auth.GetResponse{
		ID:       apiToken.ID.String(),
		Username: "token:" + apiToken.Name,
		Email:    "token@heimdallr.local",
		Roles:    roles,
	}, nil
}

func (s service) HasScope(user auth.GetResponse, scope string) bool {
	if slices.Contains(user.Roles, auth.RoleAdmin) {
		return true
	}

	return slices.Contains(user.Roles, scope)
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

func mapEntityToResponse(token APIToken) GetResponse {
	return GetResponse{
		ID:        token.ID,
		Name:      token.Name,
		Scopes:    token.Scopes,
		CreatedBy: token.CreatedBy,
	}
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
