package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth"
)

const (
	authHeaderUsername = "X-Auth-Username"
	authHeaderPassword = "X-Auth-Password"
)

func Authentication(service auth.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		username := ctx.GetHeader(authHeaderUsername)
		password := ctx.GetHeader(authHeaderPassword)

		if username == "" || password == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": auth.ErrInvalidAuthHeader.Error()})
			return
		}

		user, err := service.Authenticate(ctx.Request.Context(), username, password)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": auth.ErrInvalidCredentials.Error()})
			return
		}

		ctx.Set("auth.user", user)
		ctx.Next()
	}
}
