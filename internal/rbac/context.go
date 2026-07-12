package rbac

import (
	"github.com/gin-gonic/gin"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
)

const contextUserKey = "auth.user"

func userFromGinContext(ctx *gin.Context) (authapi.AuthUser, error) {
	value, exists := ctx.Get(contextUserKey)
	if !exists {
		return authapi.AuthUser{}, ErrUnauthorized
	}

	user, ok := value.(authapi.AuthUser)
	if !ok {
		return authapi.AuthUser{}, ErrUnauthorized
	}

	return user, nil
}
