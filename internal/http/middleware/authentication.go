package middleware

import (
	"crypto/subtle"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/token"
)

const authHeaderBearer = "Authorization"

type AuthenticationConfig struct {
	SessionCookieName string
	CSRFCookieName    string
	OptionalPaths     []string
}

func Authentication(tokenService token.Service, configs ...AuthenticationConfig) gin.HandlerFunc {
	cfg := AuthenticationConfig{
		SessionCookieName: "heimdallr_session",
		CSRFCookieName:    "heimdallr_csrf",
	}
	if len(configs) > 0 {
		cfg.OptionalPaths = slices.Clone(configs[0].OptionalPaths)
		if strings.TrimSpace(configs[0].SessionCookieName) != "" {
			cfg.SessionCookieName = configs[0].SessionCookieName
		}
		if strings.TrimSpace(configs[0].CSRFCookieName) != "" {
			cfg.CSRFCookieName = configs[0].CSRFCookieName
		}
	}

	return func(ctx *gin.Context) {
		var (
			credential string
			mechanism  auth.AuthenticationMechanism
			user       authapi.AuthUser
			err        error
		)

		bearerToken := auth.ExtractBearerToken(ctx.GetHeader(authHeaderBearer))
		if bearerToken != "" {
			user, err = tokenService.Authenticate(ctx.Request.Context(), bearerToken)
			if err == nil {
				credential = bearerToken
				mechanism = auth.AuthenticationBearer
			}
		}

		if mechanism == "" {
			sessionToken, cookieErr := ctx.Cookie(cfg.SessionCookieName)
			if cookieErr == nil && sessionToken != "" {
				user, err = tokenService.AuthenticateSession(ctx.Request.Context(), sessionToken)
				if err == nil {
					credential = sessionToken
					mechanism = auth.AuthenticationCookie
				}
			}
		}

		optionalPath := slices.Contains(cfg.OptionalPaths, ctx.Request.URL.Path)
		if mechanism == "" {
			if optionalPath {
				ctx.Next()
				return
			}

			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": auth.ErrInvalidCredentials.Error()})
			return
		}

		if mechanism == auth.AuthenticationCookie && isUnsafeMethod(ctx.Request.Method) {
			if !validCSRF(ctx, cfg.CSRFCookieName) {
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid csrf token"})
				return
			}
		}

		ctx.Set("auth.user", user)
		requestCtx := auth.ContextWithAuthentication(ctx.Request.Context(), user, mechanism, credential)
		ctx.Request = ctx.Request.WithContext(requestCtx)
		ctx.Next()
	}
}

func validCSRF(ctx *gin.Context, cookieName string) bool {
	csrfCookie, cookieErr := ctx.Cookie(cookieName)
	csrfHeader := ctx.GetHeader("X-CSRF-Token")
	return cookieErr == nil && csrfCookie != "" && csrfHeader != "" &&
		subtle.ConstantTimeCompare([]byte(csrfCookie), []byte(csrfHeader)) == 1
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}
