package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
	"gorm.io/gorm"
)

type SessionTokenCreateRequest struct {
	Name   string
	Scopes []string
}

type SessionTokenCreateResponse struct {
	Token string
}

type TokenRepository interface {
	DeleteByCreatedBy(ctx context.Context, userID string) error
	DeleteSessionTokensByCreatedBy(ctx context.Context, userID string) error
	DeleteSessionTokensByCreatedByExceptCredential(ctx context.Context, userID string, excludePlainToken string) error
	WithTx(tx *gorm.DB) TokenRepository
}

type APITokenService interface {
	Authenticate(ctx context.Context, plainToken string) (api.AuthUser, error)
	CreateSession(ctx context.Context, req SessionTokenCreateRequest, createdBy uuid.UUID) (SessionTokenCreateResponse, error)
	RevokeSessionTokens(ctx context.Context, userID string) error
	RevokeAllUserTokens(ctx context.Context, userID string) error
	RevokeSessionToken(ctx context.Context, plainToken string) error
}
