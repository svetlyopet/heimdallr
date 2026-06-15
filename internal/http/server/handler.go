package server

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/constants"
	"github.com/svetlyopet/heimdallr/internal/http/middleware"
	"github.com/svetlyopet/heimdallr/internal/logger"
)

func NewHandler(host string, appLogger *logger.Logger) (*gin.Engine, error) {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	handler := gin.New()

	handler.Use(middleware.Log(appLogger))
	handler.Use(middleware.Recover(appLogger))

	err := handler.SetTrustedProxies(constants.AppDefaultTrustedProxies)
	if err != nil {
		return nil, err
	}

	// Set up Security Headers
	handler.Use(func(c *gin.Context) {
		requestHost := c.Request.Host
		if requestHost == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing host header"})
			return
		}

		requestHostName := requestHost
		if h, _, err := net.SplitHostPort(requestHost); err == nil {
			requestHostName = h
		}
		if requestHostName != host {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid host header"})
			return
		}
		c.Header("X-Frame-Options", "DENY")
		cspPolicy := "default-src 'self'; " +
			"connect-src 'self'; " +
			"script-src 'self'; " +
			"script-src-elem 'self'; " +
			"style-src 'self'; " +
			"style-src-elem 'self'; " +
			"img-src 'self' data:; " +
			"font-src 'self'; " +
			"object-src 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'; " +
			"frame-ancestors 'none';"
		c.Header("Content-Security-Policy", cspPolicy)
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Header("Referrer-Policy", "strict-origin")
		c.Header("X-Content-Type-Options", "nosniff")
		permPolicy := "geolocation=(),midi=(),sync-xhr=(),microphone=(),camera=()," +
			"magnetometer=(),gyroscope=(),fullscreen=(self),payment=()"
		c.Header("Permissions-Policy", permPolicy)
		c.Next()
	})

	return handler, nil
}
