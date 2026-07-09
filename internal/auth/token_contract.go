package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
)

type SessionTokenCreateRequest struct {
	Name   string
	Scopes []string
}

type SessionTokenCreateResponse struct {
	Token string
}

type APITokenService interface {
	Authenticate(ctx context.Context, plainToken string) (api.AuthUser, error)
	Create(ctx context.Context, req SessionTokenCreateRequest, createdBy *uuid.UUID) (SessionTokenCreateResponse, error)
}
