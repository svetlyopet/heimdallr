package token

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/svetlyopet/heimdallr/internal/auth"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/token/api"
)

type authTokenServiceAdapter struct {
	service Service
}

func (a authTokenServiceAdapter) Authenticate(ctx context.Context, plainToken string) (authapi.AuthUser, error) {
	return a.service.Authenticate(ctx, plainToken)
}

func (a authTokenServiceAdapter) Create(ctx context.Context, req auth.SessionTokenCreateRequest, createdBy *uuid.UUID) (auth.SessionTokenCreateResponse, error) {
	scopes := make([]api.TokenScope, 0, len(req.Scopes))
	for _, scope := range req.Scopes {
		scopes = append(scopes, api.TokenScope(scope))
	}

	created, err := a.service.Create(ctx, api.TokenCreateRequest{
		Name:   req.Name,
		Scopes: scopes,
	}, createdBy)
	if err != nil {
		return auth.SessionTokenCreateResponse{}, err
	}

	return auth.SessionTokenCreateResponse{Token: created.Token}, nil
}

func provideAuthTokenService(i do.Injector) (auth.APITokenService, error) {
	return authTokenServiceAdapter{service: do.MustInvoke[Service](i)}, nil
}
