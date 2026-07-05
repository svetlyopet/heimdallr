package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/token"
)

const (
	authHeaderUsername = "X-Auth-Username"
	authHeaderPassword = "X-Auth-Password"
	authHeaderBearer   = "Authorization"
)

func Authentication(authService auth.Service, tokenService token.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if bearerToken := extractBearerToken(ctx.GetHeader(authHeaderBearer)); bearerToken != "" {
			user, err := tokenService.Authenticate(ctx.Request.Context(), bearerToken)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": auth.ErrInvalidCredentials.Error()})
				return
			}

			ctx.Set("auth.user", user)
			ctx.Next()
			return
		}

		username := ctx.GetHeader(authHeaderUsername)
		password := ctx.GetHeader(authHeaderPassword)

		if username == "" || password == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": auth.ErrInvalidAuthHeader.Error()})
			return
		}

		user, err := authService.Authenticate(ctx.Request.Context(), username, password)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": auth.ErrInvalidCredentials.Error()})
			return
		}

		ctx.Set("auth.user", user)
		ctx.Next()
	}
}

func extractBearerToken(headerValue string) string {
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
