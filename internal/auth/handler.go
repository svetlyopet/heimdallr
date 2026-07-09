package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
)

type Handler interface {
	api.StrictServerInterface
}

type handler struct {
	service      Service
	tokenService APITokenService
}

func (h handler) Login(ctx context.Context, request api.LoginRequestObject) (api.LoginResponseObject, error) {
	if request.Body == nil {
		return api.Login400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	user, err := h.service.Authenticate(ctx, request.Body.Username, request.Body.Password)
	if err != nil {
		return api.Login401JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse{Error: "invalid credentials"},
		}, nil
	}

	userID, err := uuid.Parse(user.Id)
	if err != nil {
		return api.Login500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: "invalid user id"},
		}, nil
	}

	created, err := h.tokenService.Create(ctx, SessionTokenCreateRequest{
		Name:   "session-" + user.Username,
		Scopes: loginScopesForRoles(rolesFromSlice(user.Roles)),
	}, &userID)
	if err != nil {
		return api.Login500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: "failed to create session token"},
		}, nil
	}

	return api.Login200JSONResponse{
		Data: api.LoginResponse{Token: created.Token},
	}, nil
}

func (h handler) ListUsers(ctx context.Context, _ api.ListUsersRequestObject) (api.ListUsersResponseObject, error) {
	if _, err := h.requireAdmin(ctx); err != nil {
		return adminListUsersError(err)
	}

	users, err := h.service.List(ctx)
	if err != nil {
		return api.ListUsers500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: authErrorMessage(err, "failed to list users")},
		}, nil
	}

	return api.ListUsers200JSONResponse{Data: users}, nil
}

func (h handler) CreateUser(ctx context.Context, request api.CreateUserRequestObject) (api.CreateUserResponseObject, error) {
	if _, err := h.requireAdmin(ctx); err != nil {
		return adminCreateUserError(err)
	}

	if request.Body == nil {
		return api.CreateUser400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	user, err := h.service.Create(ctx, *request.Body)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserAlreadyExists):
			return api.CreateUser409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: authErrorMessage(err, "user already exists")},
			}, nil
		case errors.Is(err, ErrInvalidRole), errors.Is(err, ErrInvalidPasswordValue), errors.Is(err, ErrInvalidCredentials):
			return api.CreateUser400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: authErrorMessage(err, err.Error())},
			}, nil
		}

		return api.CreateUser500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: authErrorMessage(err, "failed to create user")},
		}, nil
	}

	return api.CreateUser201JSONResponse{Data: user}, nil
}

func (h handler) UpdateUser(ctx context.Context, request api.UpdateUserRequestObject) (api.UpdateUserResponseObject, error) {
	if _, err := h.requireAdmin(ctx); err != nil {
		return adminUpdateUserError(err)
	}

	if request.Body == nil {
		return api.UpdateUser400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	user, err := h.service.Update(ctx, request.UserId.String(), *request.Body)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			return api.UpdateUser404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: authErrorMessage(err, "user not found")},
			}, nil
		case errors.Is(err, ErrInvalidRole), errors.Is(err, ErrInvalidPasswordValue), errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInvalidUserID), errors.Is(err, ErrRootRoleForbidden):
			return api.UpdateUser400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: authErrorMessage(err, err.Error())},
			}, nil
		}

		return api.UpdateUser500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: authErrorMessage(err, "failed to update user")},
		}, nil
	}

	return api.UpdateUser200JSONResponse{Data: user}, nil
}

func (h handler) DeleteUser(ctx context.Context, request api.DeleteUserRequestObject) (api.DeleteUserResponseObject, error) {
	if _, err := h.requireAdmin(ctx); err != nil {
		return adminDeleteUserError(err)
	}

	if err := h.service.Delete(ctx, request.UserId.String()); err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			return api.DeleteUser404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: authErrorMessage(err, "user not found")},
			}, nil
		case errors.Is(err, ErrInvalidUserID), errors.Is(err, ErrRootDeleteForbidden):
			return api.DeleteUser400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: authErrorMessage(err, err.Error())},
			}, nil
		}

		return api.DeleteUser500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: authErrorMessage(err, "failed to delete user")},
		}, nil
	}

	return api.DeleteUser204Response{}, nil
}

func NewHandler(service Service, tokenService APITokenService) (Handler, error) {
	return &handler{
		service:      service,
		tokenService: tokenService,
	}, nil
}

func (h handler) requireAdmin(ctx context.Context) (api.AuthUser, error) {
	gctx, ok := GinContextFrom(ctx)
	if !ok {
		return api.AuthUser{}, ErrInvalidCredentials
	}

	bearerToken := ExtractBearerToken(gctx.GetHeader("Authorization"))
	if bearerToken == "" {
		return api.AuthUser{}, ErrInvalidCredentials
	}

	user, err := h.tokenService.Authenticate(ctx, bearerToken)
	if err != nil {
		return api.AuthUser{}, ErrInvalidCredentials
	}

	if !h.service.HasAnyRole(user, RoleAdmin) {
		return api.AuthUser{}, ErrInsufficientRole
	}

	return user, nil
}

func loginScopesForRoles(roles []string) []string {
	for _, role := range roles {
		if role == RoleAdmin {
			return []string{"admin"}
		}
	}

	return []string{"read"}
}

func authErrorMessage(err error, fallback string) string {
	if authErr, ok := errors.AsType[AuthError](err); ok {
		return authErr.Message
	}

	return fallback
}

func adminListUsersError(err error) (api.ListUsersResponseObject, error) {
	if errors.Is(err, ErrInsufficientRole) {
		return api.ListUsers403JSONResponse{
			ForbiddenJSONResponse: api.ForbiddenJSONResponse{Error: ErrInsufficientRole.Error()},
		}, nil
	}

	return api.ListUsers401JSONResponse{
		UnauthorizedJSONResponse: api.UnauthorizedJSONResponse{Error: ErrInvalidCredentials.Error()},
	}, nil
}

func adminCreateUserError(err error) (api.CreateUserResponseObject, error) {
	if errors.Is(err, ErrInsufficientRole) {
		return api.CreateUser403JSONResponse{
			ForbiddenJSONResponse: api.ForbiddenJSONResponse{Error: ErrInsufficientRole.Error()},
		}, nil
	}

	return api.CreateUser401JSONResponse{
		UnauthorizedJSONResponse: api.UnauthorizedJSONResponse{Error: ErrInvalidCredentials.Error()},
	}, nil
}

func adminUpdateUserError(err error) (api.UpdateUserResponseObject, error) {
	if errors.Is(err, ErrInsufficientRole) {
		return api.UpdateUser403JSONResponse{
			ForbiddenJSONResponse: api.ForbiddenJSONResponse{Error: ErrInsufficientRole.Error()},
		}, nil
	}

	return api.UpdateUser401JSONResponse{
		UnauthorizedJSONResponse: api.UnauthorizedJSONResponse{Error: ErrInvalidCredentials.Error()},
	}, nil
}

func adminDeleteUserError(err error) (api.DeleteUserResponseObject, error) {
	if errors.Is(err, ErrInsufficientRole) {
		return api.DeleteUser403JSONResponse{
			ForbiddenJSONResponse: api.ForbiddenJSONResponse{Error: ErrInsufficientRole.Error()},
		}, nil
	}

	return api.DeleteUser401JSONResponse{
		UnauthorizedJSONResponse: api.UnauthorizedJSONResponse{Error: ErrInvalidCredentials.Error()},
	}, nil
}
