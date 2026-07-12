package token

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/svetlyopet/heimdallr/internal/auth"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
)

type authTokenServiceAdapter struct {
	service Service
}

func (a authTokenServiceAdapter) Authenticate(ctx context.Context, plainToken string) (authapi.AuthUser, error) {
	return a.service.Authenticate(ctx, plainToken)
}

func (a authTokenServiceAdapter) CreateSession(ctx context.Context, req auth.SessionTokenCreateRequest, createdBy uuid.UUID) (auth.SessionTokenCreateResponse, error) {
	created, err := a.service.CreateSession(ctx, req.Name, req.Scopes, createdBy)
	if err != nil {
		return auth.SessionTokenCreateResponse{}, err
	}

	return auth.SessionTokenCreateResponse{Token: created.Token}, nil
}

func (a authTokenServiceAdapter) RevokeSessionTokens(ctx context.Context, userID string) error {
	return a.service.RevokeSessionTokens(ctx, userID)
}

func (a authTokenServiceAdapter) RevokeAllUserTokens(ctx context.Context, userID string) error {
	return a.service.RevokeAllUserTokens(ctx, userID)
}

func (a authTokenServiceAdapter) RevokeSessionToken(ctx context.Context, plainToken string) error {
	return a.service.RevokeSessionToken(ctx, plainToken)
}

func provideAuthTokenService(i do.Injector) (auth.APITokenService, error) {
	return authTokenServiceAdapter{service: do.MustInvoke[Service](i)}, nil
}
