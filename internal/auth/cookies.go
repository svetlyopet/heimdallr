package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
)

type CookieConfig struct {
	SessionCookieName string
	CSRFCookieName    string
	Secure            bool
	SessionTTL        time.Duration
}

func SessionCookieMiddleware(cfg CookieConfig) api.StrictMiddlewareFunc {
	cfg = normalizeCookieConfig(cfg)

	return func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return func(ctx *gin.Context, request interface{}) (interface{}, error) {
			response, err := next(ctx, request)
			if err != nil {
				return response, err
			}

			switch operationID {
			case "Login":
				loginResponse, ok := response.(api.Login200JSONResponse)
				if !ok {
					return response, nil
				}

				csrfToken, tokenErr := generateCSRFToken()
				if tokenErr != nil {
					return nil, tokenErr
				}
				setSessionCookies(ctx, cfg, loginResponse.Data.Token, csrfToken)
			case "Logout":
				if _, ok := response.(api.Logout204Response); ok {
					clearSessionCookies(ctx, cfg)
				}
			}

			return response, nil
		}
	}
}

func normalizeCookieConfig(cfg CookieConfig) CookieConfig {
	if strings.TrimSpace(cfg.SessionCookieName) == "" {
		cfg.SessionCookieName = "heimdallr_session"
	}
	if strings.TrimSpace(cfg.CSRFCookieName) == "" {
		cfg.CSRFCookieName = "heimdallr_csrf"
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 24 * time.Hour
	}

	return cfg
}

func generateCSRFToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}

	return hex.EncodeToString(value), nil
}

func setSessionCookies(ctx *gin.Context, cfg CookieConfig, sessionToken string, csrfToken string) {
	expiresAt := time.Now().UTC().Add(cfg.SessionTTL)
	maxAge := int(cfg.SessionTTL / time.Second)
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     cfg.SessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  expiresAt,
		Secure:   cfg.Secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     cfg.CSRFCookieName,
		Value:    csrfToken,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  expiresAt,
		Secure:   cfg.Secure,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearSessionCookies(ctx *gin.Context, cfg CookieConfig) {
	expiredAt := time.Unix(1, 0).UTC()
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     cfg.SessionCookieName,
		Path:     "/",
		MaxAge:   -1,
		Expires:  expiredAt,
		Secure:   cfg.Secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     cfg.CSRFCookieName,
		Path:     "/",
		MaxAge:   -1,
		Expires:  expiredAt,
		Secure:   cfg.Secure,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})
}
