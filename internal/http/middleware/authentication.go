package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/token"
)

const authHeaderBearer = "Authorization"

func Authentication(tokenService token.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bearerToken := auth.ExtractBearerToken(ctx.GetHeader(authHeaderBearer))
		if bearerToken == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": auth.ErrInvalidAuthHeader.Error()})
			return
		}

		user, err := tokenService.Authenticate(ctx.Request.Context(), bearerToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": auth.ErrInvalidCredentials.Error()})
			return
		}

		ctx.Set("auth.user", user)
		ctx.Next()
	}
}
