package token

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/token/api"
)

type Handler interface {
	api.StrictServerInterface
}

type handler struct {
	service Service
}

func (h handler) ListTokens(ctx context.Context, _ api.ListTokensRequestObject) (api.ListTokensResponseObject, error) {
	tokens, err := h.service.List(ctx)
	if err != nil {
		return api.ListTokens500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: tokenErrorMessage(err, "failed to list tokens")},
		}, nil
	}

	return api.ListTokens200JSONResponse{Data: tokens}, nil
}

func (h handler) CreateToken(ctx context.Context, request api.CreateTokenRequestObject) (api.CreateTokenResponseObject, error) {
	if request.Body == nil {
		return api.CreateToken400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	var createdBy *uuid.UUID
	if user, userErr := auth.UserFromContext(ctx); userErr == nil {
		if parsed, parseErr := uuid.Parse(user.Id); parseErr == nil {
			createdBy = &parsed
		}
	}

	token, err := h.service.Create(ctx, *request.Body, createdBy)
	if err != nil {
		if errors.Is(err, ErrInvalidScopes) || errors.Is(err, ErrInvalidTTL) {
			return api.CreateToken400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: tokenErrorMessage(err, "invalid scopes")},
			}, nil
		}

		return api.CreateToken500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: tokenErrorMessage(err, "failed to create token")},
		}, nil
	}

	return api.CreateToken201JSONResponse{Data: token}, nil
}

func (h handler) DeleteToken(ctx context.Context, request api.DeleteTokenRequestObject) (api.DeleteTokenResponseObject, error) {
	if err := h.service.Delete(ctx, request.TokenId.String()); err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return api.DeleteToken404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: tokenErrorMessage(err, "token not found")},
			}, nil
		}

		return api.DeleteToken500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: tokenErrorMessage(err, "failed to delete token")},
		}, nil
	}

	return api.DeleteToken204Response{}, nil
}

func NewHandler(service Service) (Handler, error) {
	return &handler{service: service}, nil
}

func tokenErrorMessage(err error, fallback string) string {
	if tokenErr, ok := errors.AsType[TokenError](err); ok {
		return tokenErr.Message
	}

	return fallback
}
