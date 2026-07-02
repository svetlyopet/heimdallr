package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	authHeaderUsername = "X-Auth-Username"
	authHeaderPassword = "X-Auth-Password"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, service Service) {
	authRoutesV1 := rg.Group("/v1/auth")
	authRoutesV1.Use(RequireRoles(service, RoleAdmin))
	{
		authRoutesV1.GET("/users", handler.List)
		authRoutesV1.POST("/users", handler.Create)
		authRoutesV1.PUT("/users/:user_id", handler.Update)
		authRoutesV1.DELETE("/users/:user_id", handler.Delete)
	}
}

func RequireRoles(service Service, roles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if user, err := userFromContext(ctx); err == nil {
			if !service.HasAnyRole(user, roles...) {
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": ErrInsufficientRole.Error()})
				return
			}

			ctx.Next()
			return
		}

		username := ctx.GetHeader(authHeaderUsername)
		password := ctx.GetHeader(authHeaderPassword)

		if username == "" || password == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrInvalidAuthHeader.Error()})
			return
		}

		user, err := service.Authenticate(ctx.Request.Context(), username, password)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrInvalidCredentials.Error()})
			return
		}

		if !service.HasAnyRole(user, roles...) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": ErrInsufficientRole.Error()})
			return
		}

		ctx.Set("auth.user", user)
		ctx.Next()
	}
}

func userFromContext(ctx *gin.Context) (GetResponse, error) {
	value, exists := ctx.Get("auth.user")
	if !exists {
		return GetResponse{}, ErrInvalidCredentials
	}

	user, ok := value.(GetResponse)
	if !ok {
		return GetResponse{}, errors.New("invalid auth user context type")
	}

	return user, nil
}
