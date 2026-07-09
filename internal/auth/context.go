package auth

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
)

const contextUserKey = "auth.user"

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

func GinContextFrom(ctx context.Context) (*gin.Context, bool) {
	gctx, ok := ctx.(*gin.Context)
	return gctx, ok
}
