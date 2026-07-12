package auth

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
)

const contextUserKey = "auth.user"

type AuthenticationMechanism string

const (
	AuthenticationBearer AuthenticationMechanism = "bearer"
	AuthenticationCookie AuthenticationMechanism = "cookie"
)

type requestContextKey uint8

const (
	requestUserKey requestContextKey = iota
	requestAuthenticationMechanismKey
	requestCredentialKey
)

func UserFromGinContext(ctx *gin.Context) (api.AuthUser, error) {
	value, exists := ctx.Get(contextUserKey)
	if !exists {
		return api.AuthUser{}, ErrInvalidCredentials
	}

	user, ok := value.(api.AuthUser)
	if !ok {
		return api.AuthUser{}, ErrInvalidCredentials
	}

	return user, nil
}

func ExtractBearerToken(headerValue string) string {
	headerValue = strings.TrimSpace(headerValue)
	if headerValue == "" {
		return ""
	}

	parts := strings.SplitN(headerValue, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func ContextWithAuthentication(
	ctx context.Context,
	user api.AuthUser,
	mechanism AuthenticationMechanism,
	credential string,
) context.Context {
	ctx = context.WithValue(ctx, requestUserKey, user)
	ctx = context.WithValue(ctx, requestAuthenticationMechanismKey, mechanism)
	return context.WithValue(ctx, requestCredentialKey, credential)
}

func UserFromContext(ctx context.Context) (api.AuthUser, error) {
	if ctx == nil {
		return api.AuthUser{}, ErrInvalidCredentials
	}

	user, ok := ctx.Value(requestUserKey).(api.AuthUser)
	if !ok {
		return api.AuthUser{}, ErrInvalidCredentials
	}

	return user, nil
}

func AuthenticationFromContext(ctx context.Context) (AuthenticationMechanism, string, error) {
	if ctx == nil {
		return "", "", ErrInvalidCredentials
	}

	mechanism, mechanismOK := ctx.Value(requestAuthenticationMechanismKey).(AuthenticationMechanism)
	credential, credentialOK := ctx.Value(requestCredentialKey).(string)
	if !mechanismOK || !credentialOK || credential == "" {
		return "", "", ErrInvalidCredentials
	}

	return mechanism, credential, nil
}
